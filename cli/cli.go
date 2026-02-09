package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"memci/agent"
	"memci/config"
	memcicontext "memci/context"
	"memci/llm"
	"memci/logger"
)

// ANSI 颜色代码
const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
	Bold   = "\033[1m"
)

// CLI 表示命令行交互界面
type CLI struct {
	agent  *agent.Agent
	logger logger.Logger
	reader *bufio.Reader
}

// NewCLI 创建一个新的 CLI 实例
func NewCLI(cfg *config.Config, lg logger.Logger) *CLI {
	// 创建上下文管理器
	ctxMgr, restored := memcicontext.NewContextManager(&cfg.Context)
	if restored {
		lg.Info("Restore successfully")
		goto here
	}
	if err := ctxMgr.Initialize(); err != nil {
		lg.Fatal("Failed to initialize context manager", logger.Err(err))
	}
	// 构建系统提示词（使用 ContextManager 直接构建，绕过权限检查）
	if err := agent.BuildSystemPrompts(ctxMgr); err != nil {
		lg.Fatal("Failed to build system prompts", logger.Err(err))
	}
here:
	// 创建 Agent
	agt := agent.NewAgent(cfg, lg, llm.ModelName(cfg.LLM.AgentModel), ctxMgr)

	return &CLI{
		agent:  agt,
		logger: lg,
		reader: bufio.NewReader(os.Stdin),
	}
}

// Run 启动 CLI 交互循环
func (c *CLI) Run() error {
	c.printWelcome()

	for {
		// 读取用户输入
		input, err := c.readInput()
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		// 处理特殊命令
		if c.handleCommand(input) {
			continue
		}

		// 执行 Agent
		if err := c.executeAgent(input); err != nil {
			c.printError(err)
		}
	}
}

// printWelcome 打印欢迎界面
func (c *CLI) printWelcome() {
	fmt.Println()
	fmt.Printf("%s╔════════════════════════════════════════════════════════════════╗%s\n", Cyan, Reset)
	fmt.Printf("%s║%s                                                                %s║%s\n", Cyan, Bold, Cyan, Reset)
	fmt.Printf("%s║%s                    Memci Agent System                        %s║%s\n", Cyan, Yellow, Cyan, Reset)
	fmt.Printf("%s║%s                                                                %s║%s\n", Cyan, Bold, Cyan, Reset)
	fmt.Printf("%s╚════════════════════════════════════════════════════════════════╝%s\n", Cyan, Reset)
	fmt.Println()
	fmt.Printf("%sVersion:%s 1.0.0    %sMode:%s Interactive    %sModel:%s \n", Gray, Reset, Gray, Reset, Gray, Reset)
	fmt.Println()
	fmt.Printf("%s可用命令:%s\n", Gray, Reset)
	fmt.Printf("  %s/help%s    - 显示帮助信息\n", Yellow, Reset)
	fmt.Printf("  %s/quit%s   - 退出程序\n", Yellow, Reset)
	fmt.Printf("  %s/clear%s  - 清空屏幕\n", Yellow, Reset)
	fmt.Println()
	fmt.Printf("%s────────────────────────────────────────────────────────────────%s\n", Gray, Reset)
	fmt.Println()
}

// readInput 读取用户输入
func (c *CLI) readInput() (string, error) {
	fmt.Printf("%s◆ You:%s ", Green, Reset)
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// handleCommand 处理特殊命令，返回 true 表示是命令不需要执行 Agent
func (c *CLI) handleCommand(input string) bool {
	switch input {
	case "":
		return true
	case "/quit", "/exit", "/q":
		fmt.Printf("\n%s👋 再见！%s\n", Yellow, Reset)
		os.Exit(0)
		return true
	case "/clear", "/cls":
		c.clearScreen()
		c.printWelcome()
		return true
	case "/help", "/h":
		c.printHelp()
		return true
	}

	if strings.HasPrefix(input, "/") {
		fmt.Printf("%s⚠  未知命令: %s%s\n", Yellow, input, Reset)
		fmt.Printf("%s输入 /help 查看可用命令%s\n", Gray, Reset)
		return true
	}

	return false
}

// executeAgent 执行 Agent
func (c *CLI) executeAgent(input string) error {
	fmt.Printf("%s🔄 正在思考...%s\n", Blue, Reset)
	fmt.Println()

	result, err := c.agent.Run(nil, input)
	if err != nil {
		return fmt.Errorf("agent execution failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("agent execution failed: %s", result.Error.Error())
	}

	// 打印结果
	c.printAgentResult(result)

	return nil
}

// printAgentResult 打印 Agent 结果
func (c *CLI) printAgentResult(result *agent.AgentResult) {
	fmt.Printf("%s────────────────────────────────────────────────────────────────%s\n", Gray, Reset)
	fmt.Printf("%s◆ Agent:%s %s\n", Purple, Reset, result.FinalMessage)
	fmt.Printf("%s────────────────────────────────────────────────────────────────%s\n", Gray, Reset)

	if result.Metrics != nil {
		fmt.Printf("%s📊 指标:%s 迭代次数=%d, 工具调用=%d/%d\n",
			Gray, Reset,
			result.Iterations,
			result.Metrics.SuccessfulToolCalls,
			result.Metrics.TotalToolCalls,
		)
	}
	fmt.Println()
}

// printError 打印错误信息
func (c *CLI) printError(err error) {
	fmt.Printf("\n%s❌ 错误: %v%s\n\n", Red, err, Reset)
}

// printHelp 打印帮助信息
func (c *CLI) printHelp() {
	fmt.Println()
	fmt.Printf("%s╭──────────────────────────────────────────────────────────╮%s\n", Cyan, Reset)
	fmt.Printf("%s│%s                       %s帮助信息%s                        %s│%s\n", Cyan, Bold, Yellow, Cyan, Bold, Reset)
	fmt.Printf("%s╰──────────────────────────────────────────────────────────╯%s\n", Cyan, Reset)
	fmt.Println()
	fmt.Printf("%s特殊命令:%s\n", Gray, Reset)
	fmt.Printf("  %s/help%s    - 显示此帮助信息\n", Yellow, Reset)
	fmt.Printf("  %s/quit%s   - 退出程序\n", Yellow, Reset)
	fmt.Printf("  %s/clear%s  - 清空屏幕\n", Yellow, Reset)
	fmt.Println()
	fmt.Printf("%s交互方式:%s\n", Gray, Reset)
	fmt.Printf("  直接输入您的问题或指令，Agent 将使用工具来帮助您。\n")
	fmt.Printf("  Agent 可以管理上下文中的 Pages 和 Segments。\n")
	fmt.Println()
	fmt.Printf("%s示例:%s\n", Gray, Reset)
	fmt.Printf("  %s列出所有 Segment%s\n", Cyan, Reset)
	fmt.Printf("  %s创建一个新的详情页，名称为\"测试\"%s\n", Cyan, Reset)
	fmt.Printf("  %s显示 sys segment 的根页面信息%s\n", Cyan, Reset)
	fmt.Println()
}

// clearScreen 清空屏幕
func (c *CLI) clearScreen() {
	fmt.Print("\033[H\033[2J")
}
