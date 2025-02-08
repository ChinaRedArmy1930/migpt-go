package config

import (
	"github.com/spf13/viper"
)

type LLMConfig struct {
	LLM struct {
		APIKey      string `mapstructure:"api_key"`
		Model       string `mapstructure:"model"`
		BaseUrl     string `mapstructure:"base_url"`
		MaxTokens   int
		Temperature float32
	}
}

func LoadConfig(path string) (*LLMConfig, error) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg LLMConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
