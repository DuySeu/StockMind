package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"stockmind/internal/common"
	"stockmind/internal/database"
	core "stockmind/internal/llm"
	"stockmind/internal/llm/prompts"

	"github.com/google/uuid"
)

type chatRequest struct {
	Content   string    `json:"content"`
	SessionId uuid.UUID `json:"session_id,omitempty"`
}

func (s *Server) ChatHandler(w http.ResponseWriter, r *http.Request) {
	// ──────── Parse request ────────
	var content string
	var sessionID uuid.UUID
	var files []database.Attachment

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
			http.Error(w, "failed to create conversation: "+err.Error(), http.StatusInternalServerError)
			return
		}
		sessionID = id
	}

	// ──────── SSE headers ────────
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	if err := http.NewResponseController(w).EnableFullDuplex(); err != nil {
		log.Printf("sse: EnableFullDuplex: %v", err)
	}

	w.WriteHeader(http.StatusOK)
	common.FlushSSE(w)
	common.WriteSSE(w, common.SSEEvent("start", map[string]any{"session_id": sessionID}))

	// ──────── Load history (skip for new sessions) ────────
	var history []database.Message
	var convSummary database.ConversationSummary

	if !newSession {
		row, err := s.queries.GetConversationWithMessages(r.Context(), database.GetConversationWithMessagesParams{
			ID:     sessionID,
			Limit:  20,
			Offset: 0,
		})
		if err != nil {
			common.WriteSSE(w, common.SSEEvent(database.EventError, map[string]any{"message": err.Error()}))
			return
		}
		if len(row.ConvMetadata) > 0 {
			_ = json.Unmarshal(row.ConvMetadata, &convSummary)
		}
		if err := json.Unmarshal(row.Messages, &history); err != nil {
			common.WriteSSE(w, common.SSEEvent(database.EventError, map[string]any{"message": err.Error()}))
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
		common.WriteSSE(w, common.SSEEvent(database.EventError, map[string]any{"message": err.Error()}))
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
	go s.persistMessage(sessionID, "user", content, userMeta)

	// ──────── Call LLM ────────
	streamCh, err := s.agent.LLMChat(r.Context(), history, core.LLMOptions{SystemPrompt: systemPrompt})
	if err != nil {
		common.WriteSSE(w, common.SSEEvent(database.EventError, map[string]any{"message": err.Error()}))
		return
	}

	// ──────── Relay SSE events ────────
	var turnText strings.Builder
	toolsMap := make(map[string]*database.Tool)

	for ev := range streamCh {
		switch ev.Type {
		case database.EventText:
			turnText.WriteString(ev.Content)
		case database.EventToolCall:
			tc := ev.Data.(database.Tool)
			toolsMap[tc.ID] = &tc
		case database.EventToolResult:
			tr := ev.Data.(database.Tool)
			if t, ok := toolsMap[tr.ID]; ok {
				t.Result = tr.Result
				t.IsError = tr.IsError
			}
		}

		if ev.Type == database.EventDone {
			common.WriteSSE(w, common.SSEEvent(database.EventDone, map[string]any{"session_id": sessionID}))
		} else {
			common.WriteSSE(w, common.SSEEvent(ev.Type, ev.Data))
		}
	}

	// ──────── Persist assistant message (background) ────────
	finalText := turnText.String()
	go func() {
		var tools []database.Tool
		for _, t := range toolsMap {
			tools = append(tools, *t)
		}
		var assistMeta []database.Metadata
		if len(tools) > 0 {
			assistMeta = []database.Metadata{{Tool: tools}}
		}
		s.persistMessage(sessionID, "assistant", finalText, assistMeta)
		if !newSession {
			s.maybeSummarize(sessionID, convSummary)
		}
	}()
}

// ──────── Chat Helpers ────────

// persistMessage saves a message to DB. Uses context.Background() so it's safe
// to call as fire-and-forget in a goroutine after the request ends.
func (s *Server) persistMessage(sessionID uuid.UUID, role, content string, meta []database.Metadata) {
	if content == "" && len(meta) == 0 {
		return
	}
	log.Printf("bg: persistMessage start [session=%s role=%s]", sessionID, role)
	if err := s.queries.CreateMessage(context.Background(), database.CreateMessageParams{
		ID:             uuid.New(),
		ConversationID: sessionID,
		Role:           role,
		Content:        content,
		Metadata:       meta,
	}); err != nil {
		log.Printf("chat: save %s message: %v", role, err)
	}
	log.Printf("bg: persistMessage done [session=%s role=%s]", sessionID, role)
}

// maybeSummarize checks if summarization threshold is crossed and triggers async summarization.
func (s *Server) maybeSummarize(sessionID uuid.UUID, convSummary database.ConversationSummary) {
	log.Printf("bg: maybeSummarize start [session=%s]", sessionID)
	count, err := s.queries.GetMessageCountByConversationID(context.Background(), sessionID)
	if err != nil {
		log.Printf("chat: count messages: %v", err)
		return
	}
	if count < convSummary.SummarizedCount+core.SummarizationThreshold {
		log.Printf("bg: maybeSummarize done [session=%s] (threshold not reached)", sessionID)
		return
	}

	batch, err := s.queries.GetMessagesByConversationID(context.Background(), database.GetMessagesByConversationIDParams{
		ConversationID: sessionID,
		Limit:          int32(core.SummarizationThreshold),
		Offset:         int32(convSummary.SummarizedCount),
	})
	if err != nil {
		log.Printf("chat: fetch summarization batch: %v", err)
		return
	}

	result, err := s.agent.Summarize(batch, convSummary)
	if err != nil {
		log.Printf("chat: summarize: %v", err)
		return
	}

	metaBytes, err := json.Marshal(result)
	if err != nil {
		log.Printf("chat: marshal summary: %v", err)
		return
	}
	if err := s.queries.UpdateConversationMetadata(context.Background(), database.UpdateConversationMetadataParams{
		ID:       sessionID,
		Metadata: metaBytes,
	}); err != nil {
		log.Printf("chat: update conversation metadata: %v", err)
	}
	log.Printf("bg: maybeSummarize done [session=%s] (summarized)", sessionID)
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
				log.Printf("resolve attachment %s: %v", a.Path, err)
				return
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				log.Printf("read attachment %s: %v", a.Path, err)
				return
			}
			a.Data = data
		}(job.mi, job.ai)
	}
	wg.Wait()
}
