# Conversation Summary & Pagination Design

## Goal

Reduce LLM token usage and improve frontend performance by:
1. Auto-summarizing conversation history at every 20-message threshold
2. Paginating message loading in the frontend (scroll-to-top)
3. Using summary + last 20 messages as LLM context instead of full history

## Data Model

### conversations.metadata JSONB

```json
{
  "summary": "Condensed narrative context of all messages before the last 20...",
  "key_facts": [
    "User is analyzing VNM stock",
    "Prefers Piotroski over Z-Score",
    "Budget is 500M VND"
  ],
  "summarized_count": 40
}
```

- `summary` — single rolling narrative summary, replaced at each threshold (20, 40, 60...)
- `key_facts` — extracted discrete facts (user preferences, decisions, specific data points) that persist across re-summarizations to prevent information loss
- `summarized_count` — total messages incorporated into the summary

## Backend: LLM Chat Flow

When `LLMService.Chat()` is called:

1. Load conversation metadata (summary + key_facts + summarized_count)
2. Fetch only the last 20 messages from DB
3. Build LLM message array:
   - If summary exists: prepend system message with summary and key facts:
     ```
     Previous conversation context:
     {summary}

     Key facts from this conversation:
     - {fact_1}
     - {fact_2}
     ...
     ```
   - Append the last 20 messages as normal
4. Proceed with agentic tool loop as usual

After the assistant response completes:

1. Count total messages in the conversation
2. Check if `count >= summarized_count + 20`
3. If yes, fire a goroutine:
   - Fetch messages from `summarized_count + 1` to the new threshold
   - Include previous summary and key_facts as context if they exist
   - Call LLM with summarization prompt (returns JSON with summary + key_facts)
   - Parse response and update `conversations.metadata` with new summary, key_facts, and `summarized_count`

## Backend: Paginated Messages API

Modified endpoint: `GET /v1/sessions/{id}`

Query parameters:
- `limit` — max messages to return (default 20, max 50)
- `offset` — number of messages to skip from the end (default 0)

Response:
```json
{
  "messages": [...],
  "has_more": true
}
```

### SQL Changes

`GetMessagesByConversationID` becomes paginated:
```sql
-- name: GetMessagesByConversationID :many
SELECT * FROM (
  SELECT * FROM messages
  WHERE conversation_id = $1
  ORDER BY created_at DESC
  LIMIT $2 OFFSET $3
) sub ORDER BY created_at ASC;
```

New query:
```sql
-- name: GetMessageCountByConversationID :one
SELECT COUNT(*) FROM messages WHERE conversation_id = $1;
```

New query:
```sql
-- name: UpdateConversationMetadata :exec
UPDATE conversations SET metadata = $2 WHERE id = $1;
```

## Frontend: Scroll-to-Top Pagination

1. Initial load: fetch last 20 messages + `has_more` flag
2. Scroll detection: when `scrollTop === 0` and `has_more`, fetch next 20 older messages
3. Prepend older messages at the top of the list
4. Preserve scroll position after prepending (calculate height diff)
5. Show spinner at top while loading
6. Stop when `has_more === false`

Summary is never exposed to the frontend.

## Summarization Prompt

```
You are a conversation summarizer. Given the conversation below, produce two outputs:

1. SUMMARY: A concise context paragraph (under 500 words) that captures the narrative flow, topics discussed, and conclusions reached.

2. KEY FACTS: A bullet list of discrete, important facts extracted from the conversation. Include: user preferences, decisions made, specific data points, stock symbols discussed, analysis results, and any commitments or action items. Merge with existing key facts — update changed ones, remove obsolete ones, keep still-relevant ones.

Previous summary (if any):
{existing_summary}

Existing key facts (if any):
{existing_key_facts}

New messages to incorporate:
{messages_batch}

Respond in this exact JSON format:
{
  "summary": "...",
  "key_facts": ["fact 1", "fact 2", ...]
}
```

## Error Handling

- Summarization LLM call fails: log error, don't update metadata. Next threshold crossing retries naturally.
- Conversation deleted mid-summarization: DB update fails silently, no harm.
- Race condition (two goroutines): `summarized_count` check before update acts as guard.
- No retry logic — failed summaries picked up on next threshold.

## Changes Required

### Schema/Queries
- Modify `GetMessagesByConversationID` to accept limit/offset
- Add `GetMessageCountByConversationID`
- Add `UpdateConversationMetadata`
- Add `GetConversationByID` (to load metadata)

### Backend (Go)
- `internal/llm/service.go` — modify `Chat()` to load summary, use last 20 messages, trigger async summarization
- `internal/llm/summarizer.go` (new) — summarization logic + prompt
- `internal/server/session.handler.go` — add pagination params, return `has_more`
- `internal/database/` — regenerate with `sqlc generate`

### Frontend (React)
- `api/sessions.ts` — add limit/offset params to messages fetch
- `pages/ChatbotPage.tsx` — scroll-to-top detection, prepend messages, preserve scroll position
