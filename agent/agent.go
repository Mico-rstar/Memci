package agent

import (
	"context"
	"fmt"
	"time"

	"memci/config"
	memcicontext "memci/context"
	"memci/llm"
	"memci/logger"
	"memci/message"
	"memci/tools"
	"memci/util"
)

// Agent represents an AI agent with tool-using capabilities
type Agent struct {
	// Core components
	model         *llm.Model
	compactModel  *llm.CompactModel
	contextMgr    *memcicontext.ContextManager
	toolProvider  *tools.ContextToolsProvider
	executor      *tools.Executor

	// Configuration
	config *config.AgentConfig

	// State management
	stateManager *StateManager

	// Current turn messages (buffered before committing to context)
	currentTurnMessages *message.MessageList

	// Dialog turn counter (persists across multiple Run() calls)
	currentDialogTurn int

	// Context management state
	contextWarningSent bool

	// Observability
	logger logger.Logger
}

// NewAgent creates a new Agent instance
func NewAgent(
	cfg *config.Config,
	lg logger.Logger,
	modelName llm.ModelName,
	contextMgr *memcicontext.ContextManager,
) *Agent {
	// Get agent context from context manager
	agentCtx := contextMgr.GetAgentContext()

	// Create tool provider and executor
	toolProvider := tools.NewContextToolsProvider(agentCtx)
	env := toolProvider.RegisterTools()
	executor := tools.NewExecutor(env)

	// Create tool list for LLM
	toolList := tools.NewToolList()

	// Create LLM model
	model := llm.NewModel(cfg, lg, modelName, *toolList)

	// Create CompactModel for summarization
	compactModel := llm.NewCompactModel(cfg, lg)

	return &Agent{
		model:              model,
		compactModel:       compactModel,
		contextMgr:         contextMgr,
		toolProvider:       toolProvider,
		executor:           executor,
		config:             config.DefaultAgentConfig(),
		stateManager:       NewStateManager(),
		currentTurnMessages: message.NewMessageList(),
		logger:             lg,
	}
}

// Run executes the agent's main loop with a user query
func (a *Agent) Run(ctx context.Context, userQuery string) (*AgentResult, error) {
	a.stateManager.setState(StateRunning)
	a.stateManager.reset()
	a.currentTurnMessages = message.NewMessageList() // Reset current turn buffer
	a.currentDialogTurn++                            // Increment dialog turn counter
	a.contextWarningSent = false                     // Reset context warning flag
	defer func() {
		a.stateManager.setState(StateIdle)
	}()

	// Add user query to current turn buffer (not context yet)
	a.currentTurnMessages.AddMessage(message.User, userQuery)

	// Main ReAct loop
	for a.stateManager.GetMetrics().TotalIterations < a.config.MaxIterations {
		a.stateManager.incrementIterations()
		currentTurn := a.stateManager.GetMetrics().TotalIterations
		a.logger.Info("Starting iteration",
			logger.Int("iteration", currentTurn))

		// Export ContextWindow to file for observation
		if err := a.exportContextSnapshot(currentTurn); err != nil {
			a.logger.Warn("Failed to export context snapshot", logger.Err(err))
		}

		// Check context window
		if err := a.manageContextWindow(ctx); err != nil {
			return &AgentResult{
				Success: false,
				Error:   &AgentError{Phase: "context", Err: err, Message: "context management failed"},
			}, err
		}

		// Generate messages for LLM (from persistent context + current turn buffer)
		msgList, err := a.generateFullMessageList()
		if err != nil {
			return &AgentResult{
				Success: false,
				Error:   &AgentError{Phase: "context", Err: err, Message: "failed to generate message list"},
			}, err
		}

		// Call LLM
		llmResponse, err := a.callLLM(ctx, msgList)
		if err != nil {
			if a.shouldRetry(err) {
				a.logger.Warn("LLM call failed, retrying", logger.Err(err))
				time.Sleep(a.config.RetryDelay)
				continue
			}
			return &AgentResult{
				Success: false,
				Error:   &AgentError{Phase: "llm", Err: err, Message: "LLM call failed"},
			}, err
		}

		// fmt.Println(llmResponse.Content.String())
		// Parse LLM response
		react, err := util.ParseToolCall(llmResponse.Content.String())
		if err != nil {
			return &AgentResult{
				Success: false,
				Error:   &AgentError{Phase: "parser", Err: err, Message: "failed to parse tool call"},
			}, err
		}

		// Check if tool call exists
		if react.ToolCall == nil {
			// No tool call - agent is done, commit current turn to context
			a.logger.Info("Agent completed without tool call")
			finalMsg := llmResponse.Content.String()

			// Add assistant response to current turn buffer
			a.currentTurnMessages.AddMessage(message.Assistant, finalMsg)

			// Commit current turn messages as a single detail page
			if err := a.commitCurrentTurn(); err != nil {
				a.logger.Error("Failed to commit current turn", logger.Err(err))
				return &AgentResult{
					Success: false,
					Error:   &AgentError{Phase: "context", Err: err, Message: "failed to commit current turn"},
				}, err
			}

			return &AgentResult{
				FinalMessage: finalMsg,
				Metrics:      a.getMetricsCopy(),
				Iterations:   a.stateManager.GetMetrics().TotalIterations,
				Success:      true,
			}, nil
		}

		// Execute tool call
		toolResult, err := a.executeToolCall(ctx, react)
		fmt.Println(react.ToolCall.Code)
		if err != nil {
			// Add error message to current turn buffer
			a.logger.Error("Tool execution failed", logger.Err(err))
			errorMsg := fmt.Sprintf("Tool execution failed: %v", err)
			a.currentTurnMessages.AddMessage(message.System, errorMsg)
			continue
		}

		// a.logger.Info(toolResult.Result.(string))
		// Add tool result to current turn buffer
		a.bufferToolResult(toolResult)
	}

	// Max iterations reached
	return &AgentResult{
		Success: false,
		Error:   &MaxIterationsError{Iterations: a.stateManager.GetMetrics().TotalIterations, Message: "Agent did not complete within maximum iterations"},
		Metrics: a.getMetricsCopy(),
	}, nil
}

// manageContextWindow checks and manages token limits
func (a *Agent) manageContextWindow(ctx context.Context) error {
	currentTokens, err := a.contextMgr.EstimateTokens()
	if err != nil {
		return err
	}

	maxAllowed := a.config.MaxTokens - a.config.TokenMargin

	if currentTokens > maxAllowed {
		if !a.contextWarningSent {
			// First time: send warning to agent
			a.logger.Info("Token limit exceeded, sending warning to agent",
				logger.Int("current_tokens", currentTokens),
				logger.Int("max_allowed", maxAllowed))

			warningMsg := fmt.Sprintf(
				"[系统提醒] 当前上下文已达到 %d tokens，超出限制 %d tokens。请主动折叠无关的上下文页面，使用 hide_details 工具隐藏不相关的内容。",
				currentTokens, maxAllowed,
			)
			a.currentTurnMessages.AddMessage(message.System, warningMsg)
			a.contextWarningSent = true
		} else {
			// Warning already sent but still over limit: use AutoCollapse as fallback
			a.logger.Info("Agent did not reduce context, using AutoCollapse as fallback",
				logger.Int("current_tokens", currentTokens),
				logger.Int("max_allowed", maxAllowed))

			collapsed, err := a.contextMgr.AutoCollapse(maxAllowed)
			if err != nil {
				return err
			}

			a.logger.Info("Auto-collapsed pages",
				logger.Int("count", len(collapsed)))

			// Reset warning flag after successful collapse
			a.contextWarningSent = false
		}
	} else {
		// Context is under limit, reset warning flag
		a.contextWarningSent = false
	}

	return nil
}

// callLLM calls the LLM with the current message list
func (a *Agent) callLLM(ctx context.Context, msgList *message.MessageList) (message.Message, error) {
	a.logger.Debug("Calling LLM",
		logger.Int("message_count", msgList.Len()))

	resp, err := a.model.Process(*msgList)
	if err != nil {
		return message.Message{}, err
	}

	a.logger.Debug("LLM response received",
		logger.String("content_preview", truncateString(resp.Content.String(), 100)))

	return resp, nil
}

// executeToolCall executes a tool call from the ReAct structure
func (a *Agent) executeToolCall(ctx context.Context, react *util.ReAct) (*ToolResult, error) {
	a.logger.Info("Executing tool call",
		logger.String("target", react.ToolCall.Target))

	// Execute Starlark code
	result, err := a.executor.Execute(react.ToolCall.Code)
	if err != nil {
		a.stateManager.incrementToolCalls(false)
		return &ToolResult{
			Target:  react.ToolCall.Target,
			Success: false,
			Error:   err,
		}, err
	}

	a.stateManager.incrementToolCalls(true)

	return &ToolResult{
		Target:  react.ToolCall.Target,
		Result:  result,
		Success: true,
	}, nil
}

// shouldRetry checks if an error is retryable
func (a *Agent) shouldRetry(err error) bool {
	// TODO: Implement retry logic based on error type
	return false
}

// getMetricsCopy returns a copy of the current metrics
func (a *Agent) getMetricsCopy() *Metrics {
	metrics := a.stateManager.GetMetrics()
	return &metrics
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// generateFullMessageList combines persistent context with current turn buffer
func (a *Agent) generateFullMessageList() (*message.MessageList, error) {
	// Get messages from persistent context
	contextMsgList, err := a.contextMgr.GenerateMessageList()
	if err != nil {
		return nil, err
	}

	// Combine with current turn messages
	fullMsgList := message.NewMessageList()
	fullMsgList.AddMessageList(contextMsgList)
	fullMsgList.AddMessageList(a.currentTurnMessages)

	return fullMsgList, nil
}

// commitCurrentTurn commits the current turn messages as a summarized detail page
func (a *Agent) commitCurrentTurn() error {
	// Use CompactModel to summarize current turn messages
	summaryMsg, err := a.compactModel.Process(*a.currentTurnMessages)
	if err != nil {
		return fmt.Errorf("failed to summarize turn: %w", err)
	}

	// Get usr segment root
	usrSeg, err := a.contextMgr.GetSegment("interact")
	if err != nil {
		return fmt.Errorf("failed to get interact segment: %w", err)
	}

	rootIndex := usrSeg.GetRootIndex()

	// Create a detail page with the summary
	_, err = a.contextMgr.CreateDetailPage(
		fmt.Sprintf("Turn %d", a.currentDialogTurn),
		summaryMsg.Content.String(),
		a.currentTurnMessages.Join(),
		rootIndex,
	)


	if err != nil {
		return fmt.Errorf("failed to create detail page: %w", err)
	}

	// Hidden default
	// Expand the new page with the summary
	// return a.contextMgr.ExpandDetails(pageIndex)
	return nil
}

// bufferToolResult adds a tool result to the current turn buffer (not context)
func (a *Agent) bufferToolResult(result *ToolResult) {
	formatted := formatToolResult(result)
	if result.Success {
		a.currentTurnMessages.AddMessage(message.System, formatted)
	} else {
		a.currentTurnMessages.AddMessage(message.System, fmt.Sprintf("Tool error: %s", formatted))
	}
}

// exportContextSnapshot exports the current ContextWindow to a file
func (a *Agent) exportContextSnapshot(turn int) error {
	outputDir := "./context_snapshots"

	filepath, err := a.contextMgr.ExportToFile(outputDir, turn)
	if err != nil {
		return err
	}

	a.logger.Debug("Context snapshot exported",
		logger.String("filepath", filepath),
		logger.Int("turn", turn))

	return nil
}
