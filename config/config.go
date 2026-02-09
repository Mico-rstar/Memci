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
	CompressModel string `mapstructure:"compress_model"`
	AgentModel string `mapstructure:"agent_model"`
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
	// 存储配置
	StorageBaseDir string `toml:"storage_base_dir" mapstructure:"storage_base_dir" default:"./data/storage"`
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

	// Script executor configuration
	ScriptExecutor ScriptExecutorConfig `toml:"script_executor" mapstructure:"script_executor"`
}

// ScriptExecutorConfig holds script executor configuration
type ScriptExecutorConfig struct {
	Type string `toml:"type" mapstructure:"type" default:"starlark"` // starlark, grpc

	// gRPC specific configuration
	GRPC GRPCConfig `toml:"grpc" mapstructure:"grpc"`
}

// GRPCConfig holds gRPC executor configuration
type GRPCConfig struct {
	Address string        `toml:"address" mapstructure:"address" default:"localhost:50051"`
	Timeout time.Duration `toml:"timeout" mapstructure:"timeout" default:"30s"`
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
