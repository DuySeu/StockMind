package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"stockmind/internal/common"
	"stockmind/internal/database"
	core "stockmind/internal/llm"
	"stockmind/internal/llm/prompts"

	"github.com/google/uuid"
)

type chatRequest struct {
	Content   string    `json:"content"`
	SessionId uuid.UUID `json:"session_id,omitempty"`
	MaxMode   bool      `json:"max_mode,omitempty"`
}

// sseHeartbeatInterval bounds how long the stream can go without producing a
// byte. A turn can sit silent for 10s+ while the model reasons or a tool runs,
// which is long enough for an intermediary to decide the connection is dead.
const sseHeartbeatInterval = 15 * time.Second

// chatFailure pairs a stable code with client-safe prose. Handlers used to send
// `err.Error()` straight down the wire, which is how a deleted session surfaced
// in the UI as the driver string "no rows in result set".
type chatFailure struct {
	code    string
	message string
}

var (
	failLoadHistory = chatFailure{"history_unavailable", "Could not load this conversation's history. Please try again."}
	failPrompt      = chatFailure{"prompt_unavailable", "Could not prepare the assistant for this turn. Please try again."}
	failSaveMessage = chatFailure{"save_failed", "Could not save your message. Please try again."}
	failUpstream    = chatFailure{"upstream_unavailable", "The assistant is unavailable right now. Please try again."}
	failStream      = chatFailure{"stream_failed", "The assistant stopped unexpectedly. Please try again."}
)

// turnError builds the payload for an in-stream `error` event.
func (f chatFailure) payload() map[string]any {
	return map[string]any{"message": f.message, "code": f.code}
}

func (f chatFailure) persisted() *database.TurnError {
	return &database.TurnError{Message: f.message, Code: f.code}
}

// endTurn closes a stream out: it reports the failure (if any) and always sends
// `done`, so the client has exactly one termination signal to key off rather
// than having to treat "channel closed" as an implicit end.
func endTurn(w http.ResponseWriter, sessionID uuid.UUID, fail *chatFailure, cause error) {
	if fail != nil {
		slog.Error("chat: turn failed", "session", sessionID, "code", fail.code, "cause", cause)
		_ = common.WriteSSE(w, common.SSEEvent(database.EventError, fail.payload()))
	}
	_ = common.WriteSSE(w, common.SSEEvent(database.EventDone, map[string]any{"session_id": sessionID}))
}

func (s *Server) ChatHandler(w http.ResponseWriter, r *http.Request) {
	// ──────── Parse request ────────
	var content string
	var sessionID uuid.UUID
	var files []database.Attachment
	var maxMode bool

	log.Println("maxMode", maxMode)

	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.HasPrefix(contentType, "multipart/form-data"):
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			common.WriteJSONError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
			return
		}
		content = r.FormValue("content")
		if s := r.FormValue("session_id"); s != "" {
			if id, err := uuid.Parse(s); err == nil {
				sessionID = id
			}
		}
		if m := r.FormValue("max_mode"); m != "" {
			maxMode = m == "true"
		}
		if r.MultipartForm != nil {
			for _, fh := range r.MultipartForm.File["file"] {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				data, err := io.ReadAll(f)
				f.Close()
				if err != nil {
					continue
				}
				files = append(files, database.Attachment{
					Name:      fh.Filename,
					MediaType: fh.Header.Get("Content-Type"),
					Data:      data,
				})
			}
		}

	case strings.HasPrefix(contentType, "application/json"):
		var body chatRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			common.WriteJSONError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
			return
		}
		content = body.Content
		sessionID = body.SessionId
		maxMode = body.MaxMode // TODO: handle max mode when active, run multi-agent pipeline

	default:
		common.WriteJSONError(w, http.StatusUnsupportedMediaType, "unsupported Content-Type: expected application/json or multipart/form-data")
		return
	}

	if strings.TrimSpace(content) == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "content is required")
		return
	}

	// ──────── Upload files to MinIO ────────
	var attachments []database.Attachment
	if len(files) > 0 {
		attachments = make([]database.Attachment, len(files))
		errs := make([]error, len(files))
		var wg sync.WaitGroup
		for i, f := range files {
			wg.Add(1)
			go func(i int, f database.Attachment) {
				defer wg.Done()
				key := "chat-attachments/" + uuid.New().String() + "/" + f.Name
				errs[i] = s.objectStore.Put(r.Context(), key, bytes.NewReader(f.Data), int64(len(f.Data)))
				attachments[i] = database.Attachment{
					Name:      f.Name,
					MediaType: f.MediaType,
					Path:      key,
					Data:      f.Data,
				}
			}(i, f)
		}
		wg.Wait()
		for _, e := range errs {
			if e != nil {
				http.Error(w, "failed to upload file: "+e.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// ──────── Ensure session exists ────────
	// Validated here, while a normal JSON error response is still possible.
	// Doing it after the SSE headers are written forces every failure to be an
	// in-stream `error` event, which is how a raw "no rows in result set" used
	// to reach the client for a session that had been deleted.
	var newSession bool
	if sessionID == uuid.Nil {
		newSession = true
		userID := uuid.Must(uuid.Parse("123e4567-e89b-12d3-a456-426614174000"))
		id, err := s.queries.CreateConversation(r.Context(), database.CreateConversationParams{
			ID:       uuid.New(),
			UserID:   userID,
			Title:    "New conversation",
			Metadata: []byte("{}"),
		})
		if err != nil {
			common.WriteJSONError(w, http.StatusInternalServerError, "failed to create conversation: "+err.Error())
			return
		}
		sessionID = id
	} else if !s.requireConversation(w, r, sessionID) {
		return
	}

	// ──────── SSE headers ────────
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if err := http.NewResponseController(w).EnableFullDuplex(); err != nil {
		slog.Warn("sse: EnableFullDuplex", "error", err)
	}

	w.WriteHeader(http.StatusOK)
	common.FlushSSE(w)
	common.WriteSSE(w, common.SSEEvent("start", map[string]any{"session_id": sessionID}))

	// ──────── Load history (skip for new sessions) ────────
	var history []database.Message
	var convSummary database.ConversationSummary

	// Failures below happen before the user's message is stored, so nothing is
	// persisted and a reload shows the conversation unchanged — no orphan turn.
	if !newSession {
		row, err := s.queries.GetConversationWithMessages(r.Context(), database.GetConversationWithMessagesParams{
			ID:     sessionID,
			Limit:  20,
			Offset: 0,
		})
		if err != nil {
			endTurn(w, sessionID, &failLoadHistory, err)
			return
		}
		if len(row.ConvMetadata) > 0 {
			_ = json.Unmarshal(row.ConvMetadata, &convSummary)
		}
		if err := json.Unmarshal(row.Messages, &history); err != nil {
			endTurn(w, sessionID, &failLoadHistory, err)
			return
		}
		// Resolve attachments only for history messages (current message already has data in memory).
		s.resolveAttachments(r.Context(), history)
	}

	// ──────── System prompt ────────
	loader := prompts.NewPromptLoader()
	systemPrompt, err := loader.GetSystemPrompt(prompts.SystemParams{
		Summary:  convSummary.Summary,
		KeyFacts: strings.Join(convSummary.KeyFacts, "\n- "),
	})
	if err != nil {
		endTurn(w, sessionID, &failPrompt, err)
		return
	}

	// Append user message in-memory.
	if content != "" {
		msg := database.Message{ConversationID: sessionID, Role: "user", Content: content}
		if len(attachments) > 0 {
			msg.Metadata = []database.Metadata{{Attachments: attachments}}
		}
		history = append(history, msg)
	}

	var userMeta []database.Metadata
	if len(attachments) > 0 {
		userMeta = []database.Metadata{{Attachments: attachments}}
	}

	// Stored synchronously, and it is the line that divides the two failure
	// regimes: everything before it persists nothing, everything after it owes
	// the conversation an assistant turn — even if that turn only records why it
	// failed. As a fire-and-forget goroutine this also raced the assistant write
	// for its `created_at`, which could reorder the pair on reload.
	if err := s.persistMessageCtx(r.Context(), sessionID, "user", content, userMeta); err != nil {
		endTurn(w, sessionID, &failSaveMessage, err)
		return
	}

	// ──────── Call LLM ────────
	streamCh, err := s.agent.LLMChat(r.Context(), history, core.LLMOptions{SystemPrompt: systemPrompt})
	if err != nil {
		endTurn(w, sessionID, &failUpstream, err)
		// The question is on record, so the failure has to be too.
		go s.persistMessage(sessionID, "assistant", "", []database.Metadata{{Error: failUpstream.persisted()}})
		return
	}

	// ──────── Relay SSE events ────────
	var turnText strings.Builder
	toolsMap := make(map[string]*database.Tool)
	// Call order, kept alongside the map: ranging a map is unordered, so
	// persisting straight from it scrambled the tool sequence and the reloaded
	// conversation disagreed with what the stream had shown.
	var toolOrder []string

	var turnErr *database.TurnError
	var sawDone, clientGone bool

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	// Whatever ends the loop, the producer goroutine must not be left blocked on
	// a send into a channel nobody reads. Draining a closed channel is a no-op,
	// so this is safe on the normal path too.
	defer func() {
		go func() {
			for range streamCh {
			}
		}()
	}()

relay:
	for {
		select {
		case <-r.Context().Done():
			// The client is gone. Stop relaying — every further write fails, and
			// continuing would burn tokens generating for nobody.
			clientGone = true
			break relay

		case <-heartbeat.C:
			if err := common.WriteSSEComment(w, "ping"); err != nil {
				clientGone = true
				break relay
			}

		case ev, ok := <-streamCh:
			if !ok {
				break relay
			}
			// Only genuine silence should trigger a ping.
			heartbeat.Reset(sseHeartbeatInterval)

			switch ev.Type {
			case database.EventText:
				turnText.WriteString(ev.Content)
			case database.EventToolCall:
				tc := ev.Data.(database.Tool)
				if _, seen := toolsMap[tc.ID]; !seen {
					toolOrder = append(toolOrder, tc.ID)
				}
				toolsMap[tc.ID] = &tc
			case database.EventToolResult:
				tr := ev.Data.(database.Tool)
				if t, ok := toolsMap[tr.ID]; ok {
					t.Result = tr.Result
					t.IsError = tr.IsError
				}
			}

			var werr error
			switch ev.Type {
			case database.EventDone:
				sawDone = true
				werr = common.WriteSSE(w, common.SSEEvent(database.EventDone, map[string]any{"session_id": sessionID}))
			case database.EventError:
				// The provider's raw message is a server-side detail; the client
				// gets stable prose, and the turn gets persisted as failed.
				slog.Error("chat: stream error", "session", sessionID, "cause", ev.Data)
				turnErr = failStream.persisted()
				werr = common.WriteSSE(w, common.SSEEvent(database.EventError, failStream.payload()))
			case database.EventText, database.EventThinking:
				// Text/thinking deltas carry their payload in Content, not Data.
				// The client reads the `data` field as a string for these events.
				werr = common.WriteSSE(w, common.SSEEvent(ev.Type, ev.Content))
			default:
				werr = common.WriteSSE(w, common.SSEEvent(ev.Type, ev.Data))
			}
			if werr != nil {
				clientGone = true
				break relay
			}
		}
	}

	// The service returns without emitting `done` after an error, and a closed
	// channel is not a signal the client can see. Guarantee the terminator.
	if !clientGone && !sawDone {
		_ = common.WriteSSE(w, common.SSEEvent(database.EventDone, map[string]any{"session_id": sessionID}))
	}

	// ──────── Persist assistant message (background) ────────
	finalText := turnText.String()
	go func() {
		tools := make([]database.Tool, 0, len(toolOrder))
		for _, id := range toolOrder {
			tools = append(tools, *toolsMap[id])
		}
		var assistMeta []database.Metadata
		if len(tools) > 0 || turnErr != nil {
			assistMeta = []database.Metadata{{Tool: tools, Error: turnErr}}
		}
		s.persistMessage(sessionID, "assistant", finalText, assistMeta)
		if !newSession {
			s.maybeSummarize(sessionID, convSummary)
		}
	}()
}

// ──────── Chat Helpers ────────

// persistMessageCtx saves a message to DB under the caller's context and reports
// failure, for the paths that must not proceed on a lost write.
func (s *Server) persistMessageCtx(ctx context.Context, sessionID uuid.UUID, role, content string, meta []database.Metadata) error {
	if content == "" && len(meta) == 0 {
		return nil
	}
	return s.queries.CreateMessage(ctx, database.CreateMessageParams{
		ID:             uuid.New(),
		ConversationID: sessionID,
		Role:           role,
		Content:        content,
		Metadata:       meta,
	})
}

// persistMessage saves a message to DB. Uses context.Background() so it's safe
// to call as fire-and-forget in a goroutine after the request ends.
func (s *Server) persistMessage(sessionID uuid.UUID, role, content string, meta []database.Metadata) {
	slog.Debug("bg: persistMessage start", "session", sessionID, "role", role)
	if err := s.persistMessageCtx(context.Background(), sessionID, role, content, meta); err != nil {
		slog.Error("chat: save message", "role", role, "error", err)
	}
	slog.Debug("bg: persistMessage done", "session", sessionID, "role", role)
}

// maybeSummarize checks if summarization threshold is crossed and triggers async summarization.
func (s *Server) maybeSummarize(sessionID uuid.UUID, convSummary database.ConversationSummary) {
	slog.Debug("bg: maybeSummarize start", "session", sessionID)
	count, err := s.queries.GetMessageCountByConversationID(context.Background(), sessionID)
	if err != nil {
		slog.Error("chat: count messages", "error", err)
		return
	}
	if count < convSummary.SummarizedCount+core.SummarizationThreshold {
		slog.Debug("bg: maybeSummarize done (threshold not reached)", "session", sessionID)
		return
	}

	batch, err := s.queries.GetMessagesByConversationID(context.Background(), database.GetMessagesByConversationIDParams{
		ConversationID: sessionID,
		Limit:          int32(core.SummarizationThreshold),
		Offset:         int32(convSummary.SummarizedCount),
	})
	if err != nil {
		slog.Error("chat: fetch summarization batch", "error", err)
		return
	}

	result, err := s.agent.Summarize(batch, convSummary)
	if err != nil {
		slog.Error("chat: summarize", "error", err)
		return
	}

	metaBytes, err := json.Marshal(result)
	if err != nil {
		slog.Error("chat: marshal summary", "error", err)
		return
	}
	if err := s.queries.UpdateConversationMetadata(context.Background(), database.UpdateConversationMetadataParams{
		ID:       sessionID,
		Metadata: metaBytes,
	}); err != nil {
		slog.Error("chat: update conversation metadata", "error", err)
	}
	slog.Debug("bg: maybeSummarize done (summarized)", "session", sessionID)
}

// resolveAttachments fetches attachment data from object store for messages
// that have a Path but no Data. Modifies history in-place.
func (s *Server) resolveAttachments(ctx context.Context, history []database.Message) {
	type fetchJob struct{ mi, ai int }
	var jobs []fetchJob
	for i := range history {
		if history[i].Role != "user" || len(history[i].Metadata) == 0 {
			continue
		}
		for j := range history[i].Metadata[0].Attachments {
			a := &history[i].Metadata[0].Attachments[j]
			if a.Path == "" || a.Data != nil {
				continue
			}
			jobs = append(jobs, fetchJob{i, j})
		}
	}
	if len(jobs) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func(mi, ai int) {
			defer wg.Done()
			a := &history[mi].Metadata[0].Attachments[ai]
			rc, err := s.objectStore.Get(ctx, a.Path)
			if err != nil {
				slog.Error("resolve attachment", "path", a.Path, "error", err)
				return
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				slog.Error("read attachment", "path", a.Path, "error", err)
				return
			}
			a.Data = data
		}(job.mi, job.ai)
	}
	wg.Wait()
}
