package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	openai "github.com/sashabaranov/go-openai"
)

func createOpenAIClient(config OpenAIConfig) (*LLMClientWrapper, error) {
	var openaiClient *openai.Client

	if strings.Contains(config.BaseURL, "openrouter.ai") {
		var defaultConfig openai.ClientConfig
		key := config.APIKey
		if key == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY is not found")
		}
		defaultConfig = openai.DefaultConfig(key)
		defaultConfig.BaseURL = config.BaseURL

		openaiClient = openai.NewClientWithConfig(defaultConfig)
	} else {
		openaiClient = openai.NewClient(config.APIKey)
	}
	return &LLMClientWrapper{OfOpenAI: openaiClient}, nil
}

func createAnthropicClient(ctx context.Context, cfg AnthropicConfig) (*LLMClientWrapper, error) {
	var ac anthropic.Client
	if strings.Contains(cfg.BaseURL, "openrouter.ai") {
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key is required")
		}
		ac = anthropic.NewClient(option.WithAPIKey(cfg.APIKey), option.WithBaseURL(cfg.BaseURL))
	} else {
		defaultAWSCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(cfg.AWS.Region))
		if err != nil {
			return nil, fmt.Errorf("Failed to load AWS config")
		}
		awsCfg := defaultAWSCfg
		if cfg.AWS.Type == "assume_role" {
			if cfg.AWS.RoleARN == "" {
				return nil, fmt.Errorf("Role ARN is required for assume_role type")
			}
			awsCfg, err = config.LoadDefaultConfig(ctx,
				config.WithRegion(cfg.AWS.Region),
				config.WithCredentialsProvider(stscreds.NewAssumeRoleProvider(
					sts.NewFromConfig(defaultAWSCfg),
					cfg.AWS.RoleARN,
					func(o *stscreds.AssumeRoleOptions) {
						if cfg.AWS.Duration != 0 {
							o.Duration = time.Second * time.Duration(cfg.AWS.Duration)
						}
						if cfg.AWS.RoleSessionName != "" {
							o.RoleSessionName = cfg.AWS.RoleSessionName
						}
					},
				)),
			)
			if err != nil {
				return nil, fmt.Errorf("Failed to assume AWS role")
			}
		}
		ac = anthropic.NewClient(bedrock.WithConfig(awsCfg))
	}
	return &LLMClientWrapper{OfAnthropic: &ac}, nil
}
