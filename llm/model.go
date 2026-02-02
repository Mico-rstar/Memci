package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"memci/config"
	"memci/logger"
	"memci/message"
	"memci/tools"
	"net/http"
)

const (
	ModelQwenMax   ModelName = "qwen-max"
	ModelQwenPlus  ModelName = "qwen-plus"
	ModelQwenFlash ModelName = "qwen-flash"
)

type ModelName string

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []message.Message       `json:"messages"`
	Tools    []tools.Tool            `json:"tools,omitempty"`
}

// PromptTokensDetails contains details about prompt tokens
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	TotalTokens          int                  `json:"total_tokens"`
	PromptTokens         int                  `json:"prompt_tokens"`
	CompletionTokens     int                  `json:"completion_tokens"`
	PromptTokensDetails  PromptTokensDetails  `json:"prompt_tokens_details,omitempty"`
}

// Choice represents a completion choice
type Choice struct {
	Index   int                `json:"index"`
	Message message.Message    `json:"message"`
}

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Model represents an LLM model client
type Model struct {
	client  *http.Client
	baseURL string
	apiKey  string
	lg      logger.Logger
	name    ModelName
	tools   tools.ToolList
}

func NewModel(cfg *config.Config, logger logger.Logger, name ModelName, tools tools.ToolList) *Model {
	return &Model{
		client: &http.Client{},
		baseURL: cfg.LLM.BaseUrl,
		apiKey:  cfg.LLM.ApiKey,
		lg:      logger,
		name:    name,
		tools:   tools,
	}
}

func (m *Model) Process(msgs message.MessageList) (message.Message, error) {
	reqBody := ChatCompletionRequest{
		Model:    string(m.name),
		Messages: msgs.Msgs,
		Tools:    m.tools.ConvertToOaiFormat(),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		m.lg.Error(err.Error(), logger.F("position", "llm.Model.Process"))
		return message.Message{}, err
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		m.baseURL + "/chat/completions",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		m.lg.Error(err.Error(), logger.F("position", "llm.Model.Process"))
		return message.Message{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.client.Do(req)

	if err != nil {
		m.lg.Error(err.Error(), logger.F("position", "llm.Model.Process"))
		return message.Message{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		m.lg.Error(err.Error(), logger.F("position", "llm.Model.Process"))
		return message.Message{}, err
	}

	if resp.StatusCode != http.StatusOK {
		m.lg.Error(string(body), logger.F("position", fmt.Sprintf("llm.Model.Process status=%d", resp.StatusCode)))
		return message.Message{}, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var rsp ChatCompletionResponse
	if err := json.Unmarshal(body, &rsp); err != nil {
		m.lg.Error(err.Error(), logger.F("position", "llm.Model.Process"))
		return message.Message{}, err
	}


	if len(rsp.Choices) == 0 {
		m.lg.Error("no response", logger.F("position", "llm.Model.Process"))
		return message.Message{}, fmt.Errorf("no choices in response")
	}

	msg := message.Message{
		Role:      message.Assistant,
		Content:   rsp.Choices[0].Message.Content,
		ToolCalls: rsp.Choices[0].Message.ToolCalls,
	}

	fmt.Printf("Total Tokens: %d\nCached Tokens: %d\n", rsp.Usage.TotalTokens, rsp.Usage.PromptTokensDetails.CachedTokens)

	return msg, nil
}
