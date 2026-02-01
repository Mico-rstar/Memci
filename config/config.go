package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	LLM LLMConfig `mapstructure:"llm"`
	Log LogConfig `mapstructure:"log"`
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
