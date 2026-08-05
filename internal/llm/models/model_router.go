package models

import (
	"encoding/base64"
	"encoding/json"
	"stockmind/internal/database"
	"strings"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"
	anthropic "github.com/anthropics/anthropic-sdk-go"
	openai "github.com/openai/openai-go/v3"
)

// DBToAnthropicMessages maps stored conversation rows into Anthropic message params,
// merging consecutive same-role messages as the API requires.
func DBToAnthropicMessages(messages []database.Message) ([]anthropic.MessageParam, error) {
	var result []anthropic.MessageParam

	for _, m := range messages {
		var blocks []anthropic.ContentBlockParamUnion
		content := strings.TrimSpace(m.Content)
		if content != "" {
			blocks = append(blocks, anthropic.ContentBlockParamUnion{
				OfText: &anthropic.TextBlockParam{Text: content},
			})
		}

		// Append image blocks for user messages with attachments.
		if m.Role == "user" && len(m.Metadata) > 0 {
			for _, a := range m.Metadata[0].Attachments {
				if strings.HasPrefix(a.MediaType, "image/") {
					blocks = append(blocks, anthropic.NewImageBlockBase64(
						a.MediaType,
						base64.StdEncoding.EncodeToString(a.Data),
					))
				}
			}
		}

		var toolResultBlocks []anthropic.ContentBlockParamUnion

		if len(m.Metadata) > 0 {
			meta := m.Metadata[0]
			for _, t := range meta.Tool {
				if t.ID != "" {
					inputJSON := json.RawMessage(t.Arguments)
					if len(inputJSON) == 0 {
						inputJSON = []byte("{}")
					}
					blocks = append(blocks, anthropic.ContentBlockParamUnion{
						OfToolUse: &anthropic.ToolUseBlockParam{
							ID:    t.ID,
							Name:  t.Name,
							Input: inputJSON,
						},
					})

					if t.Result != "" || t.IsError != "" {
						isError := t.IsError == "true"
						toolResultBlocks = append(toolResultBlocks, anthropic.ContentBlockParamUnion{
							OfToolResult: &anthropic.ToolResultBlockParam{
								ToolUseID: t.ID,
								Content: []anthropic.ToolResultBlockParamContentUnion{
									{OfText: &anthropic.TextBlockParam{Text: t.Result}},
								},
								IsError: anthropic.Bool(isError),
							},
						})
					}
				}
			}
		}

		if len(blocks) > 0 {
			role := anthropic.MessageParamRoleUser
			if m.Role == "assistant" {
				role = anthropic.MessageParamRoleAssistant
			}
			result = append(result, anthropic.MessageParam{
				Role:    role,
				Content: blocks,
			})
		}

		if len(toolResultBlocks) > 0 {
			result = append(result, anthropic.MessageParam{
				Role:    anthropic.MessageParamRoleUser,
				Content: toolResultBlocks,
			})
		}
	}

	var merged []anthropic.MessageParam
	for _, msg := range result {
		if len(merged) > 0 && merged[len(merged)-1].Role == msg.Role {
			merged[len(merged)-1].Content = append(merged[len(merged)-1].Content, msg.Content...)
		} else {
			merged = append(merged, msg)
		}
	}

	return merged, nil
}

// DBToOpenAIMessages maps stored conversation rows into OpenAI chat message params,
// emitting a tool message after each assistant round that called tools.
func DBToOpenAIMessages(messages []database.Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	var result []openai.ChatCompletionMessageParamUnion

	for _, m := range messages {
		content := strings.TrimSpace(m.Content)

		switch m.Role {
		case "user":
			var attachments []database.Attachment
			if len(m.Metadata) > 0 {
				attachments = m.Metadata[0].Attachments
			}
			if len(attachments) > 0 {
				parts := []openai.ChatCompletionContentPartUnionParam{
					openai.TextContentPart(content),
				}
				for _, a := range attachments {
					if strings.HasPrefix(a.MediaType, "image/") {
						dataURL := "data:" + a.MediaType + ";base64," + base64.StdEncoding.EncodeToString(a.Data)
						parts = append(parts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
							URL: dataURL,
						}))
					}
				}
				result = append(result, openai.UserMessage(parts))
			} else {
				result = append(result, openai.UserMessage(content))
			}
		case "assistant":
			msg := openai.ChatCompletionAssistantMessageParam{
				Content: openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(content),
				},
			}

			var toolResults []openai.ChatCompletionMessageParamUnion

			if len(m.Metadata) > 0 {
				meta := m.Metadata[0]
				for _, t := range meta.Tool {
					if t.ID != "" {
						msg.ToolCalls = append(msg.ToolCalls, openai.ChatCompletionMessageToolCallUnionParam{
							OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
								ID: t.ID,
								Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
									Name:      t.Name,
									Arguments: t.Arguments,
								},
							},
						})

						toolResults = append(toolResults, openai.ToolMessage(t.Result, t.ID))
					}
				}
			}

			result = append(result, openai.ChatCompletionMessageParamUnion{
				OfAssistant: &msg,
			})
			result = append(result, toolResults...)
		default:
			result = append(result, openai.UserMessage(content))
		}
	}

	return result, nil
}

// DBToOpenRouterMessages maps stored conversation rows into OpenRouter chat messages.
func DBToOpenRouterMessages(messages []database.Message) ([]components.ChatMessages, error) {
	msgs := make([]components.ChatMessages, 0, len(messages))

	appendToolResultMessages := func(meta []database.Metadata) {
		if len(meta) == 0 {
			return
		}
		for _, t := range meta[0].Tool {
			if t.ID == "" {
				continue
			}
			msgs = append(msgs, components.CreateChatMessagesTool(components.ChatToolMessage{
				Role:       components.ChatToolMessageRoleTool,
				ToolCallID: t.ID,
				Content:    components.CreateChatToolMessageContentStr(t.Result),
			}))
		}
	}

	for _, m := range messages {
		var msg components.ChatMessages
		role := strings.ToLower(m.Role)
		content := strings.TrimSpace(m.Content)

		switch role {
		case "user":
			var attachments []database.Attachment
			if len(m.Metadata) > 0 {
				attachments = m.Metadata[0].Attachments
			}
			if len(attachments) > 0 {
				parts := []components.ChatContentItems{
					components.CreateChatContentItemsText(components.ChatContentText{Text: content}),
				}
				for _, a := range attachments {
					if strings.HasPrefix(a.MediaType, "image/") {
						dataURL := "data:" + a.MediaType + ";base64," + base64.StdEncoding.EncodeToString(a.Data)
						parts = append(parts, components.CreateChatContentItemsImageURL(components.ChatContentImage{
							ImageURL: components.ChatContentImageImageURL{URL: dataURL},
						}))
					}
				}
				msg = components.CreateChatMessagesUser(components.ChatUserMessage{
					Role:    components.ChatUserMessageRoleUser,
					Content: components.CreateChatUserMessageContentArrayOfChatContentItems(parts),
				})
			} else {
				msg = components.CreateChatMessagesUser(components.ChatUserMessage{
					Role:    components.ChatUserMessageRoleUser,
					Content: components.CreateChatUserMessageContentStr(content),
				})
			}
		case "assistant":
			var toolCalls []components.ChatToolCall
			if len(m.Metadata) > 0 {
				for _, t := range m.Metadata[0].Tool {
					if t.ID != "" {
						toolCalls = append(toolCalls, components.ChatToolCall{
							ID:   t.ID,
							Type: components.ChatToolCallTypeFunction,
							Function: components.ChatToolCallFunction{
								Name:      t.Name,
								Arguments: t.Arguments,
							},
						})
					}
				}
			}
			assistantMsg := components.ChatAssistantMessage{
				Role:      components.ChatAssistantMessageRoleAssistant,
				ToolCalls: toolCalls,
			}
			if content != "" {
				assistantMsg.Content = optionalnullable.From(openrouter.Pointer(components.CreateChatAssistantMessageContentStr(content)))
			}
			msgs = append(msgs, components.CreateChatMessagesAssistant(assistantMsg))
			appendToolResultMessages(m.Metadata)
			continue
		case "tool":
			appendToolResultMessages(m.Metadata)
			continue
		}

		msgs = append(msgs, msg)
	}

	return msgs, nil
}
