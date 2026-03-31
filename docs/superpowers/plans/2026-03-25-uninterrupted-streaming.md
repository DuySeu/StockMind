# Uninterrupted Streaming Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement resilient AI chat streaming that survives page reloads by separating the generation lifecycle from the HTTP request using an in-memory event hub.

**Architecture:** We use an In-Memory Event Hub with Replay Buffer. The backend manages a `StreamManager` containing active `SessionStream`s. The `chat` endpoint spawns generation in a goroutine while a new `stream` endpoint serves the current buffer plus live events via SSE. Frontend separates chat initialization and stream connection.

**Tech Stack:** Go (Backend), React/TypeScript (Frontend), SSE.

---

### Chunk 1: Backend Infrastructure

### Task 1: Create StreamManager

**Files:**
- Create: `internal/server/stream.go`

- [ ] **Step 1: Write StreamManager implementation**

```go
package server

import (
	"stockmind/internal/agent"
	"sync"

	"github.com/google/uuid"
)

type SessionStream struct {
	events      []agent.ChatEvent
	subscribers []chan agent.ChatEvent
	mu          sync.Mutex
	isComplete  bool
}

type StreamManager struct {
	streams map[uuid.UUID]*SessionStream
	mu      sync.RWMutex
}

var GlobalStreamManager = &StreamManager{
	streams: make(map[uuid.UUID]*SessionStream),
}

func (sm *StreamManager) CreateStream(sessionID uuid.UUID) *SessionStream {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	stream := &SessionStream{
		events:      make([]agent.ChatEvent, 0),
		subscribers: make([]chan agent.ChatEvent, 0),
		isComplete:  false,
	}
	sm.streams[sessionID] = stream
	return stream
}

func (sm *StreamManager) GetStream(sessionID uuid.UUID) (*SessionStream, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	stream, exists := sm.streams[sessionID]
	return stream, exists
}

func (sm *StreamManager) RemoveStream(sessionID uuid.UUID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.streams, sessionID)
}

func (ss *SessionStream) AddEvent(event agent.ChatEvent) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	ss.events = append(ss.events, event)
	if event.IsEnd {
		ss.isComplete = true
	}
	
	for _, sub := range ss.subscribers {
		select {
		case sub <- event:
		default:
		}
	}
}

func (ss *SessionStream) Subscribe() (chan agent.ChatEvent, []agent.ChatEvent, bool) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	ch := make(chan agent.ChatEvent, 100)
	
	if !ss.isComplete {
		ss.subscribers = append(ss.subscribers, ch)
	}
	
	return ch, ss.events, ss.isComplete
}

func (ss *SessionStream) Unsubscribe(ch chan agent.ChatEvent) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	for i, sub := range ss.subscribers {
		if sub == ch {
			ss.subscribers = append(ss.subscribers[:i], ss.subscribers[i+1:]...)
			break
		}
	}
	close(ch)
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/server/stream.go
git commit -m "feat(backend): add StreamManager for in-memory chat session streaming"
```

### Task 2: Update Chat Handler & Add Stream Endpoint

**Files:**
- Modify: `internal/server/routes.go`

- [ ] **Step 1: Mount the new stream endpoint**

Modify `RegisterRoutes` in `internal/server/routes.go` around line 48:
```go
		// Websocket
		// r.Get("/ws", s.websocketHandler)
		r.Post("/chat", s.chatHandler)
		r.Get("/chat/stream", s.chatStreamHandler)
```

- [ ] **Step 2: Rewrite chatHandler to use Goroutine and StreamManager**

Replace the current `chatHandler` implementation with:
```go
func (s *Server) chatHandler(w http.ResponseWriter, r *http.Request) {
	content, sessionID, attachments, err := parseChatRequest(r)
	if err != nil {
		fmt.Printf("chatHandler parsing error: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if content == "" && len(attachments) == 0 {
		http.Error(w, "content or attachment is required", http.StatusBadRequest)
		return
	}

	userID := uuid.Must(uuid.Parse("123e4567-e89b-12d3-a456-426614174000"))
	agentID := uuid.Must(uuid.Parse("01993ca8-a62e-79e3-995c-a46e25a4a2a2"))
	var sessionIdPtr *uuid.UUID
	if sessionID != uuid.Nil {
		sessionIdPtr = &sessionID
	}

	session, err := s.agent.GetOrCreateSession(&userID, &agentID, sessionIdPtr, &content)
	if err != nil {
		fmt.Printf("Failed to get or create session: %v\n", err)
		http.Error(w, "Failed to get or create session", http.StatusInternalServerError)
		return
	}

	err = session.Initialize()
	if err != nil {
		fmt.Printf("Failed to initialize session: %v\n", err)
		http.Error(w, "Failed to initialize session", http.StatusInternalServerError)
		return
	}

	stream := GlobalStreamManager.CreateStream(session.SessionID())

	go func() {
		defer func() {
			stream.AddEvent(agent.ChatEvent{IsEnd: true})
		}()

		session.AddChatCallback(func(event agent.ChatEvent) error {
			stream.AddEvent(event)
			return nil
		})

		err = session.HumanInput(content, attachments)
		if err != nil {
			fmt.Printf("Failed to send human input: %v\n", err)
			return
		}
		
		nLoop := 0
		for !session.IsHumanTurn() {
			err = session.ContinueTurn()
			if err != nil {
				fmt.Printf("Failed to continue turn: %v\n", err)
				return
			}
			nLoop++
			if nLoop > 10 {
				fmt.Println("Too many loops (max: 10), terminating")
				return
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": session.SessionID().String(),
	})
}
```

- [ ] **Step 3: Implement chatStreamHandler**

Add `chatStreamHandler` to `internal/server/routes.go`:
```go
func (s *Server) chatStreamHandler(w http.ResponseWriter, r *http.Request) {
	sessionIDStr := r.URL.Query().Get("session_id")
	if sessionIDStr == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		http.Error(w, "invalid session_id format", http.StatusBadRequest)
		return
	}

	stream, exists := GlobalStreamManager.GetStream(sessionID)
	
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if !exists {
		// Stream might be completed or never existed. Send a complete event to be safe.
		writeSSE(w, map[string]any{"type": "complete", "data": map[string]any{"session_id": sessionID.String()}})
		return
	}

	ch, history, isComplete := stream.Subscribe()
	if !isComplete {
		defer stream.Unsubscribe(ch)
	}

	// Replay history buffer
	inThinkingBlock := false
	for _, event := range history {
		sendEventToSSE(w, sessionID.String(), event, &inThinkingBlock)
	}

	if isComplete {
		return
	}

	// Listen for new events
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			sendEventToSSE(w, sessionID.String(), event, &inThinkingBlock)
			if event.IsEnd {
				return
			}
		}
	}
}

func sendEventToSSE(w http.ResponseWriter, sessionID string, event agent.ChatEvent, inThinkingBlock *bool) {
	switch event.Type {
	case agent.EventTypeText:
		if *inThinkingBlock {
			*inThinkingBlock = false
		}
		writeSSE(w, map[string]any{"type": "text_delta", "data": map[string]any{"text": event.Content}})
	case agent.EventTypeThinking:
		if !*inThinkingBlock {
			*inThinkingBlock = true
		}
		writeSSE(w, map[string]any{"type": "thinking_delta", "data": map[string]any{"thinking": event.Content}})
	case agent.EventTypeToolUse:
		writeSSE(w, map[string]any{"type": "tool_use", "data": map[string]any{"tool_calls": event.ToolUse}})
	case agent.EventTypeToolResult:
		writeSSE(w, map[string]any{"type": "tool_result", "data": map[string]any{"result": event.ToolResult}})
	}
	if event.IsEnd {
		writeSSE(w, map[string]any{"type": "complete", "data": map[string]any{"session_id": sessionID}})
	}
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/server/routes.go
git commit -m "feat(backend): implement separate chat stream endpoint and background generation"
```

---

### Chunk 2: Frontend Integration

### Task 3: Split Frontend API Client Function

**Files:**
- Modify: `frontend/src/api/chat.ts`

- [ ] **Step 1: Split chatWithLLM into startChatSession and streamChatSession**

In `frontend/src/api/chat.ts`, replace `chatWithLLM` export:
```typescript
import api from "./index";

export interface ChatMessage {
  content: string;
  session_id?: string;
}

export interface ChatResponse {
  type: "thinking_delta" | "text_delta" | "tool_use" | "tool_result" | "complete";
  data?: {
    thinking?: string;
    text?: string;
    session_id?: string;
    tool_calls?: Record<string, any>;
    result?: Record<string, any>;
  };
}

export const startChatSession = async (
  content: string,
  sessionId: string | undefined,
  file?: File | null
): Promise<string> => {
  let body;
  const headers: Record<string, string> = {};

  if (file) {
    const formData = new FormData();
    formData.append("content", content);
    if (sessionId) formData.append("session_id", sessionId);
    formData.append("file", file);
    body = formData;
  } else {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify({ content, session_id: sessionId });
  }

  const response = await fetch(`${api.defaults.baseURL}/chat`, {
    method: "POST",
    headers,
    body,
  });

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  const data = await response.json();
  return data.session_id;
};

export const streamChatSession = async (
  sessionId: string,
  onMessage: (data: ChatResponse) => void,
  onError: (error: any) => void
) => {
  try {
    const response = await fetch(`${api.defaults.baseURL}/chat/stream?session_id=${sessionId}`, {
      method: "GET",
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    if (!response.body) {
      throw new Error("Response body is null");
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (line.startsWith("data: ")) {
          const data = line.slice(6);
          try {
            const parsed = JSON.parse(data) as ChatResponse;
            onMessage(parsed);
          } catch (e) {
            console.error("Error parsing SSE data:", e);
          }
        }
      }
    }
  } catch (error) {
    onError(error);
  }
};
```

Keep the `getMessages` export.

- [ ] **Step 2: Commit**

```bash
git add frontend/src/api/chat.ts
git commit -m "feat(frontend): separate chat API into start and stream functions"
```

### Task 4: Connect Chatbot UI

**Files:**
- Modify: `frontend/src/pages/Chatbot.tsx`

- [ ] **Step 1: Update imports in Chatbot.tsx**
```typescript
import { startChatSession, streamChatSession, getMessages } from "@/api/chat";
```

- [ ] **Step 2: Add handleStreamEvent helper inside ChatbotPage component**

Above `onSubmit`:
```typescript
  const handleStreamEvent = (data: any, targetIndex: number, sessionIdToSet?: string) => {
    switch (data.type) {
      case "thinking_delta": {
        setMessages((prev) => {
          const updated = [...prev];
          const currentMessage = updated[targetIndex] || { role: "assistant", content: [] };
          const newContent = [...(currentMessage.content || [])];
          const delta = data.data?.thinking ?? "";
          let idx = newContent.findIndex((c) => c.type === "thinking");
          if (idx === -1) {
            newContent.push({ type: "thinking", thinking: "", is_open: true });
            idx = newContent.length - 1;
          }
          const block = newContent[idx];
          if (block.type === "thinking") {
            newContent[idx] = { ...block, thinking: (block.thinking ?? "") + delta };
          }
          updated[targetIndex] = { ...currentMessage, content: newContent };
          return updated;
        });
        break;
      }
      case "tool_use": {
        const tool_calls = data.data?.tool_calls;
        if (tool_calls) {
          setMessages((prev) => {
            const updated = [...prev];
            const currentMessage = updated[targetIndex] || { role: "assistant", content: [] };
            const existingToolCalls = currentMessage.tool_calls || [];
            updated[targetIndex] = {
              ...currentMessage,
              tool_calls: [...existingToolCalls, tool_calls],
            };
            return updated;
          });
        }
        break;
      }
      case "tool_result": {
        setMessages((prev) => [...prev, data.data?.result as Message]);
        break;
      }
      case "text_delta": {
        setMessages((prev) => {
          const updated = [...prev];
          const currentMessage = updated[targetIndex] || { role: "assistant", content: [] };
          const newContent = [...(currentMessage.content || [])];
          const delta = data.data?.text ?? "";
          let idx = newContent.findIndex((c) => c.type === "text");
          if (idx === -1) {
            newContent.push({ type: "text", text: "" });
            idx = newContent.length - 1;
          }
          const block = newContent[idx];
          if (block.type === "text") {
            newContent[idx] = { ...block, text: (block.text ?? "") + delta };
          }
          updated[targetIndex] = { ...currentMessage, content: newContent };
          return updated;
        });
        break;
      }
      case "complete": {
        if (sessionIdToSet) {
          // You might have let sessionId = data.data?.session_id; outside this function.
        }
        setMessages((prev) => {
          const updated = [...prev];
          const currentMessage = updated[targetIndex] || { role: "assistant", content: [] };
          const newContent = [...(currentMessage.content || [])];
          const idx = newContent.findIndex((c) => c.type === "thinking");
          if (idx !== -1) {
            const block = newContent[idx];
            if (block.type === "thinking") {
              newContent[idx] = { ...block, is_open: false };
            }
          }
          updated[targetIndex] = { ...currentMessage, content: newContent };
          return updated;
        });
        break;
      }
    }
  };
```

- [ ] **Step 3: Update `onSubmit` and reload `useEffect`**

Replace `onSubmit`'s `chatWithLLM` call with:
```typescript
    try {
      const activeSessionId = await startChatSession(data.input.trim(), id || undefined, fileToSend);
      if (!id && activeSessionId) {
        setTitle(data.input.trim());
        navigate(`/c/${activeSessionId}`, { replace: true });
      }
      
      await streamChatSession(
        activeSessionId,
        (data) => handleStreamEvent(data, assistantIndex, activeSessionId),
        (error) => {
          console.error("Error sending message:", error);
          setMessages((prev) => {
            const updated = [...prev];
            updated[assistantIndex] = {
              role: "assistant",
              content: [{ type: "text", text: "Error sending message" }],
            };
            return updated;
          });
        }
      );
    } catch(err) {
      console.error(err);
    }
```

Replace the `useEffect` that fetches messages:
```typescript
  useEffect(() => {
    const fetchMessagesAndStream = async () => {
      if (id) {
        const loadedMessages = await getMessages(id);
        
        // Handle stream reconnect
        const isUserLast = loadedMessages.length > 0 && loadedMessages[loadedMessages.length - 1].role === "user";
        let targetIndex = loadedMessages.length;
        if (isUserLast) {
          loadedMessages.push({ role: "assistant", content: [] });
        } else {
          targetIndex = loadedMessages.length - 1;
        }
        setMessages(loadedMessages);

        await streamChatSession(
          id,
          (data) => handleStreamEvent(data, targetIndex),
          (error) => {
            console.log("Stream check finished or not active", error);
          }
        );
      } else {
        setMessages([]);
      }
    };

    fetchMessagesAndStream();
  }, [id]);
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/Chatbot.tsx
git commit -m "feat(frontend): connect chat UI to new streaming reconnect logic"
```
