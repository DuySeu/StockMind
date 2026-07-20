package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"stockmind/internal/common"
	"stockmind/internal/database"
	"stockmind/internal/llm/tools"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type AnthropicCompletionParams struct {
	Context context.Context
	Client  *anthropic.Client
	Model   string
	Prompt  string
}

// anthropicProvider adapts the Anthropic SDK glue to the Provider interface.
type anthropicProvider struct {
	client *anthropic.Client
	model  string
}

func (p *anthropicProvider) Completion(ctx context.Context, history []database.Message, toolDefs []*tools.Tool, systemPrompt string) (<-chan database.StreamEvent, error) {
	return AnthropicCompletion(AnthropicCompletionParams{
		Context: ctx,
		Client:  p.client,
		Model:   p.model,
		Prompt:  systemPrompt,
	}, history, toolDefs)
}

func (p *anthropicProvider) StructuredCompletion(ctx context.Context, prompt string, result any) error {
	return AnthropicStructuredCompletion(AnthropicCompletionParams{
		Context: ctx,
		Client:  p.client,
		Model:   p.model,
		Prompt:  prompt,
	}, result)
}

// NewAnthropicClient builds an Anthropic client (direct API or AWS Bedrock) from the given config.
func NewAnthropicClient(ctx context.Context, cfg common.Anthropic) (*anthropic.Client, error) {
	var ac anthropic.Client
	if strings.Contains(cfg.BaseURL, "openrouter.ai") {
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key is required")
		}
		ac = anthropic.NewClient(
			option.WithAPIKey(cfg.APIKey),
			option.WithBaseURL(cfg.BaseURL),
			option.WithHTTPClient(common.SharedHTTPClient),
		)
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
	return &ac, nil
}

// AnthropicCompletion sends messages to the Anthropic API and returns a streaming event channel.
// It supports text streaming and tool use via the official anthropic-sdk-go streaming API.
func AnthropicCompletion(params AnthropicCompletionParams, messages []database.Message, tools []*tools.Tool) (<-chan database.StreamEvent, error) {
	ch := make(chan database.StreamEvent, 256)
	// -- Map messages --
	anthropicMsgs, err := DBToAnthropicMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("anthropic: failed to map messages: %w", err)
	}

	// -- Map tools --
	anthropicTools := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		schema, _ := json.Marshal(t.Schema)
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: schema,
				},
			},
		})
	}

	// -- Build request params --
	req := anthropic.MessageNewParams{
		Model:     anthropic.Model(params.Model),
		MaxTokens: 8096,
		Messages:  anthropicMsgs,
	}
	if params.Prompt != "" {
		req.System = []anthropic.TextBlockParam{
			{Text: params.Prompt},
		}
	}
	if len(anthropicTools) > 0 {
		req.Tools = anthropicTools
	}

	// -- Start stream --
	stream := params.Client.Messages.NewStreaming(params.Context, req)

	go func() {
		defer close(ch)

		var currentToolID string
		var currentToolName string
		var currentToolArgs string

		for stream.Next() {
			event := stream.Current()

			switch ev := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch delta := ev.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					if delta.Text != "" {
						ch <- database.StreamEvent{
							Type:    database.EventText,
							Content: delta.Text,
						}
					}
				case anthropic.InputJSONDelta:
					// Accumulate tool input JSON fragments
					currentToolArgs += delta.PartialJSON
				}

			case anthropic.ContentBlockStartEvent:
				block := ev.ContentBlock.AsAny()
				if toolUse, ok := block.(anthropic.ToolUseBlock); ok {
					currentToolID = toolUse.ID
					currentToolName = toolUse.Name
					currentToolArgs = ""
				}

			case anthropic.ContentBlockStopEvent:
				// Emit tool call when a tool_use block finishes
				if currentToolID != "" {
					ch <- database.StreamEvent{
						Type: database.EventToolCall,
						Data: database.Tool{
							ID:        currentToolID,
							Name:      currentToolName,
							Arguments: currentToolArgs,
						},
					}
					currentToolID = ""
					currentToolName = ""
					currentToolArgs = ""
				}

			case anthropic.MessageStopEvent:
				_ = ev
				ch <- database.StreamEvent{Type: database.EventDone}
				return
			}
		}

		if err := stream.Err(); err != nil {
			ch <- database.StreamEvent{Type: database.EventError, Data: err.Error()}
			return
		}

		// Fallback done signal if MessageStopEvent was not received
		ch <- database.StreamEvent{Type: database.EventDone}
	}()

	return ch, nil
}

// AnthropicStructuredCompletion calls the Anthropic API without streaming.
// The prompt must describe the desired JSON structure and instruct the model to respond with JSON only.
// Result must be a pointer; the JSON response is unmarshalled into it.
func AnthropicStructuredCompletion(params AnthropicCompletionParams, result any) error {
	req := anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(params.Model),
		MaxTokens: 4096,
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(params.Prompt)),
		},
		OutputFormat: anthropic.BetaJSONOutputFormatParam{
			Schema: result,
		},
		Betas: []anthropic.AnthropicBeta{"structured-outputs-2025-11-13"},
	}

	res, err := params.Client.Beta.Messages.New(params.Context, req)
	if err != nil {
		return fmt.Errorf("anthropic structured: %w", err)
	}
	if len(res.Content) == 0 {
		return fmt.Errorf("anthropic structured: empty response")
	}

	text := res.Content[0].Text
	if text == "" {
		return fmt.Errorf("anthropic structured: no text content")
	}
	if err := json.Unmarshal([]byte(text), result); err != nil {
		return fmt.Errorf("anthropic structured: %w", err)
	}
	return nil
}
