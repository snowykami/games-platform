package config

import (
	"encoding/json"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTP     HTTPConfig
	Database DatabaseConfig
	Redis    RedisConfig
	AI       AIConfig
	OIDC     []OIDCProviderConfig
}

type HTTPConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type AIConfig struct {
	LLMAPI   string
	LLMModel string
	LLMToken string
}

type OIDCProviderConfig struct {
	Key          string   `json:"key"`
	DisplayName  string   `json:"displayName"`
	IssuerURL    string   `json:"issuerUrl"`
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	RedirectURL  string   `json:"redirectUrl"`
	Scopes       []string `json:"scopes"`
}

func Load() Config {
	loadDotEnv()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8901"
	}

	return Config{
		HTTP:     HTTPConfig{Port: port},
		Database: DatabaseConfig{URL: os.Getenv("DB_URL")},
		Redis:    RedisConfig{URL: os.Getenv("REDIS_URL")},
		AI: AIConfig{
			LLMAPI:   os.Getenv("LLM_API"),
			LLMModel: os.Getenv("LLM_MODEL"),
			LLMToken: os.Getenv("LLM_TOKEN"),
		},
		OIDC: loadOIDCProviders(),
	}
}

func (c AIConfig) Enabled() bool {
	return c.LLMAPI != "" && c.LLMModel != "" && c.LLMToken != ""
}

func (c DatabaseConfig) Enabled() bool {
	return c.URL != ""
}

func (c RedisConfig) Enabled() bool {
	return c.URL != ""
}

func loadDotEnv() {
	// godotenv.Load follows the usual local-dev convention: it does not
	// overwrite variables already exported by the shell or deployment runtime.
	_ = godotenv.Load(".env", "../.env")
}

func loadOIDCProviders() []OIDCProviderConfig {
	raw := os.Getenv("OIDC_PROVIDERS_JSON")
	if raw == "" {
		return nil
	}

	var providers []OIDCProviderConfig
	if err := json.Unmarshal([]byte(raw), &providers); err != nil {
		return nil
	}
	return providers
}
