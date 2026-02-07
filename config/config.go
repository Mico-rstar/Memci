package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	LLM     LLMConfig      `mapstructure:"llm"`
	Log     LogConfig      `mapstructure:"log"`
	Context ContextConfig  `mapstructure:"context"`
	Agent   AgentConfig    `mapstructure:"agent"`
}

type LLMConfig struct {
	ApiKey string `mapstructure:"DASHSCOPE_API_KEY"`
	BaseUrl string `mapstructure:"DASHSCOPE_BASE_URL"`
}

type LogConfig struct {
	Level      string `toml:"level" mapstructure:"level" env:"LOG_LEVEL" default:"debug"`   // debug, info, warn, error
	Output     string `toml:"output" mapstructure:"output" env:"LOG_OUTPUT" default:"both"` // console, file, both
	FilePath   string `toml:"file_path" mapstructure:"file_path" env:"LOG_FILE_PATH" default:"logs/app.log"`
	MaxSizeMB  int    `toml:"max_size_mb" mapstructure:"max_size_mb" default:"100"`
	MaxBackups int    `toml:"max_backups" mapstructure:"max_backups" default:"3"`
	MaxAgeDays int    `toml:"max_age_days" mapstructure:"max_age_days" default:"30"`
}

// ContextConfig 上下文系统配置
type ContextConfig struct {
	// Active Chapter 配置
	ActiveMaxPages int `toml:"active_max_pages" mapstructure:"active_max_pages" default:"5"` // ActiveChapter 最多保留的 Page 数量

	// Archive Chapter 配置
	ArchiveMaxToken int `toml:"archive_max_token" mapstructure:"archive_max_token" default:"2000"` // Contents Page 最大 token 数

	// 卸载策略配置
	UnloadMinRecallTurns  int     `toml:"unload_min_recall_turns" mapstructure:"unload_min_recall_turns" default:"10"`    // m 轮对话后召回次数为 0 则卸载
	UnloadStaleRecallTurns int    `toml:"unload_stale_recall_turns" mapstructure:"unload_stale_recall_turns" default:"20"`  // p 轮对话后召回次数不变则卸载
	UnloadMaxTokenRatio    float64 `toml:"unload_max_token_ratio" mapstructure:"unload_max_token_ratio" default:"0.9"`    // 达到最大 token 的比例时按 LRU 卸载

	// 存储配置
	StorageBaseDir string `toml:"storage_base_dir" mapstructure:"storage_base_dir" default:"./data/pages"` // Page 存储目录
	StorageUseGzip bool   `toml:"storage_use_gzip" mapstructure:"storage_use_gzip" default:"true"`          // 是否使用 gzip 压缩存储
}

// AgentConfig holds agent configuration
type AgentConfig struct {
	// Loop control
	MaxIterations    int           // Maximum iterations in ReAct loop (default: 10)
	IterationTimeout time.Duration // Timeout per iteration (default: 30s)

	// Token management
	MaxTokens   int // Max tokens before auto-collapse (default: 8000)
	TokenMargin int // Safety margin for tokens (default: 1000)

	// Error handling
	MaxRetries int           // Max retries on transient errors (default: 3)
	RetryDelay time.Duration // Delay between retries (default: 1s)

	// Tool execution
	ToolTimeout time.Duration // Timeout for tool execution (default: 10s)
}

// DefaultAgentConfig returns default agent configuration
func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		MaxIterations:    10,
		IterationTimeout: 30 * time.Second,
		MaxTokens:        8000,
		TokenMargin:      1000,
		MaxRetries:       3,
		RetryDelay:       1 * time.Second,
		ToolTimeout:      10 * time.Second,
	}
}

var config Config

func LoadConfig(path string) *Config {
	v := viper.New()
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("toml")
		v.AddConfigPath(".")
	}
	
	// viper.AutomaticEnv()
	err := v.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = v.Unmarshal(&config)
	if err != nil {
		panic(err)
	}

	return &config
}
