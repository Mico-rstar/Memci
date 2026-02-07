package main

import (
	"fmt"
	"os"

	"memci/cli"
	"memci/config"
	"memci/logger"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig("config.toml")

	// 创建 logger
	lg, err := logger.New(cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// 打印启动信息
	lg.Info("Starting Memci Agent System",
		logger.String("version", "1.0.0"),
		logger.String("log_level", cfg.Log.Level))

	// 创建并运行 CLI
	c := cli.NewCLI(cfg, lg)

	if err := c.Run(); err != nil {
		lg.Error("CLI error", logger.Err(err))
		fmt.Fprintf(os.Stderr, "\n❌ Fatal error: %v\n", err)
		os.Exit(1)
	}
}
