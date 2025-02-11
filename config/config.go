package config

import (
	"github.com/spf13/viper"
)

var DefaultConfig *Config

func init() {
	DefaultConfig, _ = LoadConfig("config.yaml")
}

type LLMConfig struct {
	APIKey         string `mapstructure:"api_key"`
	Model          string `mapstructure:"model"`
	BaseUrl        string `mapstructure:"base_url"`
	EmbeddingUrl   string `mapstructure:"embedding_url"`
	EmbeddingModel string `mapstructure:"embedding_model"`
	MaxTokens      int
	Temperature    float32
}

type AiConfig struct {
	WakeUpKeyWords []string `mapstructure:"wake_up_key_words"`
}

type Config struct {
	LLM LLMConfig
	Ai  AiConfig
}

func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	DefaultConfig = &cfg
	return &cfg, nil
}
