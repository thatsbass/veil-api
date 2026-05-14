package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Port     int    `mapstructure:"PORT"`
	Env      string `mapstructure:"ENV"`

	DatabaseURL string `mapstructure:"DATABASE_URL"`
	RedisURL    string `mapstructure:"REDIS_URL"`

	DeepSeekAPIKey string `mapstructure:"DEEPSEEK_API_KEY"`

	StripeSecretKey      string `mapstructure:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret  string `mapstructure:"STRIPE_WEBHOOK_SECRET"`

	APIKeySecret string `mapstructure:"API_KEY_SECRET"`
	JWTSecret    string `mapstructure:"JWT_SECRET"`
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = viper.ReadInConfig()

	viper.SetDefault("PORT", 8080)
	viper.SetDefault("ENV", "development")

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config.Load: %w", err)
	}
	return cfg, nil
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
