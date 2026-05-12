package core

import (
	"net/http"
	"os"
	"time"

	"stockmind/internal/database"
)

type LLMProviderConfig struct {
	OpenAI     OpenAIConfig     `json:"openai" yaml:"openai,omitempty"`
	Anthropic  AnthropicConfig  `json:"anthropic" yaml:"anthropic,omitempty"`
	OpenRouter OpenRouterConfig `json:"openrouter" yaml:"openrouter,omitempty"`
}

type OpenRouterConfig struct {
	APIKey  string `json:"api_key" yaml:"api_key"`
	BaseURL string `json:"baseURL" yaml:"baseURL"`
}

type OpenAIConfig struct {
	APIKey  string `json:"api_key" yaml:"api_key"`
	BaseURL string `json:"baseURL" yaml:"baseURL"`
}

type AnthropicConfig struct {
	APIKey  string              `json:"api_key,omitempty" yaml:"api_key,omitempty"`
	BaseURL string              `json:"baseURL,omitempty" yaml:"baseURL,omitempty"`
	AWS     AWSCredentialConfig `json:"aws,omitempty" yaml:"aws,omitempty"`
}

type AWSCredentialConfig struct {
	Type            string `json:"type" yaml:"type"` // "default" or "assume_role"
	Region          string `json:"region" yaml:"region"`
	RoleARN         string `json:"roleArn,omitempty" yaml:"roleArn,omitempty"`                 // required if Type is "assume_role"
	Duration        int64  `json:"duration,omitempty" yaml:"duration,omitempty"`               // in seconds, optional
	RoleSessionName string `json:"roleSessionName,omitempty" yaml:"roleSessionName,omitempty"` // optional
}

func LoadLLMConfig() LLMProviderConfig {
	return LLMProviderConfig{
		OpenAI: OpenAIConfig{
			APIKey:  os.Getenv("OPENROUTER_API_KEY"),
			BaseURL: "https://openrouter.ai/api/v1",
		},
		Anthropic: AnthropicConfig{
			BaseURL: "https://openrouter.ai/api",
			APIKey:  os.Getenv("OPENROUTER_API_KEY"),
		},
		OpenRouter: OpenRouterConfig{
			APIKey:  os.Getenv("OPENROUTER_API_KEY"),
			BaseURL: "https://openrouter.ai/api/v1",
		},
		// Anthropic: AnthropicConfig{
		// 	AuthType: "aws",
		// 	AWS: AWSCredentialConfig{
		// 		Type:            "assume_role",
		// 		Region:          "ap-southeast-1",
		// 		RoleARN:         "arn:aws:iam::130506138320:role/bedrock-cross-account-role",
		// 		Duration:        3600,
		// 		RoleSessionName: "llm-test-session",
		// 	},
		// },
	}
}

// GetModelName returns the model name from the LLM_MODEL environment variable.
// Example: "openai/gpt-4o", "anthropic/claude-3-5-sonnet"
func GetModelName() string {
	return os.Getenv("LLM_MODEL")
}

// GetProviderName returns the provider name from the LLM_PROVIDER environment variable.
// Supported values: "openai", "anthropic"
func GetProviderName() database.ModelProvider {
	return database.ModelProvider(os.Getenv("LLM_PROVIDER"))
}

// sharedHTTPClient is a process-wide HTTP client with a tuned connection pool.
// All non-Bedrock LLM SDKs reuse it so TLS handshakes and TCP connections are
// amortized across requests, reducing time-to-first-token.
var sharedHTTPClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	},
}
