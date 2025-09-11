package core

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
)

// Model ID in OpenRouter support function calling and free (https://openrouter.ai/models)
const (
	NEMOTRON_NANO_9B_V2 = "nvidia/nemotron-nano-9b-v2:free"
	GLM_4_5_AIR         = "z-ai/glm-4.5-air:free"
	QWEN3_CODE          = "qwen/qwen3-coder:free"
	QWEN3_4B            = "qwen/qwen3-4b:free"
	QWEN3_235B          = "qwen/qwen3-235b-a22b:free"
	KIMI_K2             = "moonshotai/kimi-k2:free"
	MISTRAL_SMALL       = "mistralai/mistral-small-3.2-24b-instruct:free"
	DEVTRAL_SMALL       = "mistralai/devstral-small-2505:free"
	DEEPSEEK_V3         = "deepseek/deepseek-chat-v3-0324:free"
)

func init() {
	_ = godotenv.Load()
}

type Agent struct {
	SystemPrompt string
	ModelId      string
	MaxTokens    int
	Temperature  float32
	Tools        []openai.Tool
	Stream       bool
}

func (a *Agent) createClient() (*openai.Client, error) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY is not set")
	}
	cfg := openai.DefaultConfig(key)
	cfg.BaseURL = "https://openrouter.ai/api/v1"
	client := openai.NewClientWithConfig(cfg)
	return client, nil
}

func (a *Agent) preparePayload(message string) (openai.ChatCompletionRequest, error) {
	dialogue := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: a.SystemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: message},
	}

	req := openai.ChatCompletionRequest{
		Model:       a.ModelId,
		Messages:    dialogue,
		MaxTokens:   a.MaxTokens,
		Temperature: a.Temperature,
		Stream:      a.Stream,
	}

	if len(a.Tools) >= 1 {
		req.Tools = a.Tools
	}

	return req, nil
}

func (a *Agent) Invoke(message string) (string, error) {
	ctx := context.Background()
	client, err := a.createClient()

	if err != nil {
		return "", err
	}

	payload, err := a.preparePayload(message)
	if err != nil {
		return "", err
	}

	response, err := client.CreateChatCompletion(ctx, payload)
	if err != nil {
		return "", err
	}

	msg := response.Choices[0].Message
	if len(msg.ToolCalls) != 1 {
		fmt.Printf("No tool calls found\n")
		return response.Choices[0].Message.Content, nil
	}

	// Append the tool use to the payload
	payload.Messages = append(payload.Messages, msg)

	// Append the tool result to the payload
	payload.Messages = append(payload.Messages, openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Content:    "Sunny and 80 degrees.", // TODO: Replace with Actual tool result get from function
		Name:       msg.ToolCalls[0].Function.Name,
		ToolCallID: msg.ToolCalls[0].ID,
	})

	response, err = client.CreateChatCompletion(ctx, payload)
	if err != nil {
		return "", err
	}

	return response.Choices[0].Message.Content, nil
}

func (a *Agent) InvokeStream(message string) {

}
