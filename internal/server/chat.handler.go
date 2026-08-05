package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"stockmind/internal/common"
	"stockmind/internal/database"
	core "stockmind/internal/llm"
	"stockmind/internal/llm/prompts"
	"stockmind/internal/orchestration"

	"github.com/google/uuid"
)

// A turn can sit silent for 10s+ while the model reasons, long enough for an
// intermediary to decide the connection is dead.
const sseHeartbeatInterval = 15 * time.Second

// Bounded because it lands in a JSONB column read back on every history page.
const maxPersistedThinking = 24 * 1024

// Namespaced so a synthetic step ID can never collide with a provider's own
// tool-call IDs: both travel in the same stream, matched back by ID.
const stepToolPrefix = "step:"

const (
	pipelineGoalTurns = 10
	pipelineGoalChars = 600
)

var (
	failLoadHistory = chatFailure{"history_unavailable", "Could not load this conversation's history. Please try again."}
	failPrompt      = chatFailure{"prompt_unavailable", "Could not prepare the assistant for this turn. Please try again."}
	failSaveMessage = chatFailure{"save_failed", "Could not save your message. Please try again."}
	failUpstream    = chatFailure{"upstream_unavailable", "The assistant is unavailable right now. Please try again."}
	failStream      = chatFailure{"stream_failed", "The assistant stopped unexpectedly. Please try again."}
	failQuota       = chatFailure{"quota_exhausted", "The AI provider quota has run out. Retrying now will fail the same way — check the plan's credits or rate limit, then try again later."}
)

// Provider quota refusals reach this package only as prose, so matching text is
// the only way to tell them from a generic outage. Bare status numbers are
// deliberately absent: a request id or token count can contain "429".
var quotaSignals = []string{
	"insufficient_quota",
	"insufficient credits",
	"insufficient_credits",
	"exceeded your current quota",
	"quota exceeded",
	"credit balance is too low",
	"more credits",      // OpenRouter: "requires more credits, or fewer max_tokens"
	"rate limit",        // "Rate limit exceeded" — OpenRouter, OpenAI
	"rate_limit",        // Anthropic: rate_limit_error
	"ratelimit",         // some gateways collapse it
	"quota",             // catch-all for provider wording not listed above
	"payment required",  // 402 status text
	"too many requests", // 429 status text
}

type chatRequest struct {
	Content   string    `json:"content"`
	SessionId uuid.UUID `json:"session_id,omitempty"`
	MaxMode   bool      `json:"max_mode,omitempty"`
}

type chatInput struct {
	Content   string
	SessionID uuid.UUID
	MaxMode   bool
	Files     []database.Attachment
}

// Exists so a failed turn carries client-safe prose: handlers used to send
// `err.Error()` down the wire, surfacing "no rows in result set" in the UI.
type chatFailure struct {
	code    string
	message string
}

// POST /v1/chat - Run one conversational turn and stream it as SSE
func (s *Server) ChatHandler(w http.ResponseWriter, r *http.Request) {
	in, ok := parseChatInput(w, r)
	if !ok {
		return
	}

	attachments, err := s.storeAttachments(r.Context(), in.Files)
	if err != nil {
		common.WriteJSONError(w, http.StatusInternalServerError, "failed to upload file: "+err.Error())
		return
	}

	sessionID := in.SessionID
	// Validated before the SSE headers, while a normal JSON error is still
	// possible; after them every failure has to become an in-stream event.
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

	var history []database.Message
	var convSummary database.ConversationSummary

	// Failures below still precede the user's message being stored, so a reload
	// shows the conversation unchanged rather than an orphan turn.
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
		// History only — the current message already carries its data in memory.
		s.resolveAttachments(r.Context(), history)
	}

	loader := prompts.NewPromptLoader()
	systemPrompt, err := loader.GetSystemPrompt(prompts.SystemParams{
		Summary:  convSummary.Summary,
		KeyFacts: strings.Join(convSummary.KeyFacts, "\n- "),
	})
	if err != nil {
		endTurn(w, sessionID, &failPrompt, err)
		return
	}

	if in.Content != "" {
		msg := database.Message{ConversationID: sessionID, Role: "user", Content: in.Content}
		if len(attachments) > 0 {
			msg.Metadata = []database.Metadata{{Attachments: attachments}}
		}
		history = append(history, msg)
	}

	var userMeta []database.Metadata
	if len(attachments) > 0 {
		userMeta = []database.Metadata{{Attachments: attachments}}
	}

	// The line dividing the two failure regimes: before it nothing is persisted,
	// after it the conversation is owed an assistant turn even if that turn only
	// records why it failed. Synchronous because a goroutine raced the assistant
	// write for `created_at` and could reorder the pair on reload.
	if err := s.persistMessageCtx(r.Context(), sessionID, "user", in.Content, userMeta); err != nil {
		endTurn(w, sessionID, &failSaveMessage, err)
		return
	}

	streamCh, err := s.turnStream(r.Context(), in, history, systemPrompt, convSummary)
	if err != nil {
		fail := failureFor(err.Error(), failUpstream)
		endTurn(w, sessionID, &fail, err)
		// The question is on record, so the failure has to be too.
		go s.persistMessage(sessionID, "assistant", "", []database.Metadata{{Error: fail.persisted()}})
		return
	}

	var turnText, turnThinking strings.Builder
	toolsMap := make(map[string]*database.Tool)
	// Ranging a map is unordered, which scrambled the persisted tool sequence
	// against what the stream had shown.
	var toolOrder []string

	var turnErr *database.TurnError
	var sawDone, clientGone bool

	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()

	// However the loop ends, the producer must not be left blocked sending into a
	// channel nobody reads. Draining a closed channel is a no-op.
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
			case database.EventThinking:
				if turnThinking.Len() < maxPersistedThinking {
					turnThinking.WriteString(ev.Content)
				}
			case database.EventToolCall:
				// Data is typed any with more than one producer, so an unexpected
				// payload drops the event rather than panicking the relay.
				if tc, ok := ev.Data.(database.Tool); ok {
					if _, seen := toolsMap[tc.ID]; !seen {
						toolOrder = append(toolOrder, tc.ID)
					}
					toolsMap[tc.ID] = &tc
				}
			case database.EventToolResult:
				if tr, ok := ev.Data.(database.Tool); ok {
					if t, ok := toolsMap[tr.ID]; ok {
						t.Result = tr.Result
						t.IsError = tr.IsError
					}
				}
			}

			var werr error
			switch ev.Type {
			case database.EventDone:
				sawDone = true
				werr = common.WriteSSE(w, common.SSEEvent(database.EventDone, map[string]any{"session_id": sessionID}))
			case database.EventError:
				// The provider's raw message is a server-side detail; the client gets
				// stable prose and the turn is persisted as failed.
				slog.Error("chat: stream error", "session", sessionID, "cause", ev.Data)
				fail := failureFor(fmt.Sprint(ev.Data), failStream)
				turnErr = fail.persisted()
				werr = common.WriteSSE(w, common.SSEEvent(database.EventError, fail.payload()))
			case database.EventText, database.EventThinking:
				// These deltas carry their payload in Content, and the client reads
				// `data` as a string for them.
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
	// channel is not a signal the client can see.
	if !clientGone && !sawDone {
		_ = common.WriteSSE(w, common.SSEEvent(database.EventDone, map[string]any{"session_id": sessionID}))
	}

	finalText := turnText.String()
	thinkingText := turnThinking.String()
	go func() {
		tools := make([]database.Tool, 0, len(toolOrder))
		for _, id := range toolOrder {
			tools = append(tools, *toolsMap[id])
		}
		var assistMeta []database.Metadata
		if len(tools) > 0 || turnErr != nil || thinkingText != "" {
			assistMeta = []database.Metadata{{Tool: tools, Error: turnErr, Thinking: thinkingText}}
		}
		s.persistMessage(sessionID, "assistant", finalText, assistMeta)
		if !newSession {
			s.maybeSummarize(sessionID, convSummary)
		}
	}()
}

func isQuotaFailure(cause string) bool {
	lower := strings.ToLower(cause)
	for _, signal := range quotaSignals {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	return false
}

// A spent quota is the one upstream failure "please try again" is wrong about:
// the next attempt fails identically until credits or the rate window reset.
func failureFor(cause string, fallback chatFailure) chatFailure {
	if isQuotaFailure(cause) {
		return failQuota
	}
	return fallback
}

func (f chatFailure) payload() map[string]any {
	return map[string]any{"message": f.message, "code": f.code}
}

func (f chatFailure) persisted() *database.TurnError {
	return &database.TurnError{Message: f.message, Code: f.code}
}

// Always sends `done` so the client has exactly one termination signal, rather
// than having to treat "channel closed" as an implicit end.
func endTurn(w http.ResponseWriter, sessionID uuid.UUID, fail *chatFailure, cause error) {
	if fail != nil {
		slog.Error("chat: turn failed", "session", sessionID, "code", fail.code, "cause", cause)
		_ = common.WriteSSE(w, common.SSEEvent(database.EventError, fail.payload()))
	}
	_ = common.WriteSSE(w, common.SSEEvent(database.EventDone, map[string]any{"session_id": sessionID}))
}

// Writes its own JSON error response: every rejection here precedes the SSE
// headers, which keeps a bad request an ordinary 4xx.
func parseChatInput(w http.ResponseWriter, r *http.Request) (chatInput, bool) {
	var in chatInput
	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.HasPrefix(contentType, "multipart/form-data"):
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			common.WriteJSONError(w, http.StatusBadRequest, "invalid multipart form: "+err.Error())
			return in, false
		}
		in.Content = r.FormValue("content")
		if v := r.FormValue("session_id"); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				in.SessionID = id
			}
		}
		in.MaxMode = r.FormValue("max_mode") == "true"
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
				in.Files = append(in.Files, database.Attachment{
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
			return in, false
		}
		in.Content = body.Content
		in.SessionID = body.SessionId
		in.MaxMode = body.MaxMode

	default:
		common.WriteJSONError(w, http.StatusUnsupportedMediaType, "unsupported Content-Type: expected application/json or multipart/form-data")
		return in, false
	}

	if strings.TrimSpace(in.Content) == "" {
		common.WriteJSONError(w, http.StatusBadRequest, "content is required")
		return in, false
	}
	return in, true
}

// The first failure aborts the turn: answering against a question whose
// attachment is missing is worse than not starting.
func (s *Server) storeAttachments(ctx context.Context, files []database.Attachment) ([]database.Attachment, error) {
	if len(files) == 0 {
		return nil, nil
	}

	stored := make([]database.Attachment, len(files))
	errs := make([]error, len(files))
	var wg sync.WaitGroup
	for i, f := range files {
		wg.Add(1)
		go func(i int, f database.Attachment) {
			defer wg.Done()
			key := "chat-attachments/" + uuid.New().String() + "/" + f.Name
			errs[i] = s.objectStore.Put(ctx, key, bytes.NewReader(f.Data), int64(len(f.Data)))
			stored[i] = database.Attachment{
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
			return nil, e
		}
	}
	return stored, nil
}

// The only place the two flows diverge; both emit the same StreamEvent
// vocabulary so the caller relays and persists them one way. Attachments reach
// only the single-agent flow — an agent step is driven by the planner's text
// task, so a max-mode turn stores and shows an upload without sending it.
func (s *Server) turnStream(
	ctx context.Context,
	in chatInput,
	history []database.Message,
	systemPrompt string,
	summary database.ConversationSummary,
) (<-chan database.StreamEvent, error) {
	if in.MaxMode {
		if s.orchestrator == nil {
			// Configuration gap, not a client error: answering with the ordinary flow
			// beats failing the turn outright.
			slog.Warn("chat: max mode requested but no pipeline is configured; falling back to the single-agent flow")
		} else {
			return s.pipelineStream(ctx, pipelineRun(history, summary))
		}
	}
	// Reasoning is forwarded because this model routinely spends a whole round in
	// that channel and leaves `content` empty; dropping it left the turn with
	// nothing to show or store, and persistence declines an empty message.
	return s.agent.LLMChat(ctx, history, core.LLMOptions{SystemPrompt: systemPrompt, StreamThinking: true})
}

// An empty message is not stored at all, which is what keeps a turn that
// produced nothing from appearing as a blank bubble after a reload.
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

// Uses context.Background() so it is safe to call fire-and-forget after the
// request context is already cancelled.
func (s *Server) persistMessage(sessionID uuid.UUID, role, content string, meta []database.Metadata) {
	slog.Debug("bg: persistMessage start", "session", sessionID, "role", role)
	if err := s.persistMessageCtx(context.Background(), sessionID, role, content, meta); err != nil {
		slog.Error("chat: save message", "role", role, "error", err)
	}
	slog.Debug("bg: persistMessage done", "session", sessionID, "role", role)
}

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

// Modifies history in place.
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

// Translating into the chat vocabulary here, rather than teaching the chat path
// a second event set, is what lets ChatHandler relay, persist and summarise a
// max-mode turn with the same code it uses for an ordinary one — and what makes
// a reloaded max-mode turn render like the live stream did.
func (s *Server) pipelineStream(ctx context.Context, req orchestration.Request) (<-chan database.StreamEvent, error) {
	stream, err := s.orchestrator.Run(ctx, req)
	if err != nil {
		return nil, err
	}

	out := make(chan database.StreamEvent, 8)
	go func() {
		defer close(out)
		var lastStepID string
		for ev := range stream {
			// The client concatenates the thinking deltas it receives, so without a
			// break at each step boundary the plan and every step's prose run together
			// as one block and the markdown breaks with them.
			if ev.Type == orchestration.EventStepText {
				if ref, ok := ev.Data.(orchestration.StepRef); ok && ref.ID != lastStepID {
					lastStepID = ref.ID
					out <- database.StreamEvent{Type: database.EventThinking, Content: "\n\n### " + ref.Agent + "\n"}
				}
			}
			// A dropped event had no chat equivalent — a step's completion payload,
			// say, already reported as a tool result.
			if mapped, ok := chatEventFor(ev); ok {
				out <- mapped
			}
		}
	}()
	return out, nil
}

// Keeps the pipeline legible in a chat transcript: plan and step prose become
// thinking, each step becomes a tool chip that resolves when it finishes.
func chatEventFor(ev orchestration.Event) (database.StreamEvent, bool) {
	switch ev.Type {
	case orchestration.EventPlan:
		plan, ok := ev.Data.(orchestration.Plan)
		if !ok {
			return database.StreamEvent{}, false
		}
		return database.StreamEvent{Type: database.EventThinking, Content: renderPlan(plan)}, true

	case orchestration.EventStepStart:
		info, ok := ev.Data.(orchestration.StepInfo)
		if !ok {
			return database.StreamEvent{}, false
		}
		return database.StreamEvent{
			Type: database.EventToolCall,
			Data: database.Tool{
				ID:        stepToolPrefix + info.ID,
				Name:      info.Agent,
				Arguments: stepArguments(info),
			},
		}, true

	case orchestration.EventStepText:
		// Live progress, not the answer: the answer arrives whole as EventFinal, so
		// streaming these as `text` would print every step into the reply.
		return database.StreamEvent{Type: database.EventThinking, Content: ev.Content}, true

	case orchestration.EventStepDone, orchestration.EventStepError:
		res, ok := ev.Data.(orchestration.StepResult)
		if !ok {
			return database.StreamEvent{}, false
		}
		tool := database.Tool{ID: stepToolPrefix + res.ID, Name: res.Agent, Result: res.Content}
		if res.Err != "" {
			tool.IsError = "true"
			tool.Result = stepFailureMessage(res.Err)
		}
		return database.StreamEvent{Type: database.EventToolResult, Data: tool}, true

	case orchestration.EventToolCall:
		return database.StreamEvent{Type: database.EventToolCall, Data: ev.Data}, true

	case orchestration.EventToolResult:
		return database.StreamEvent{Type: database.EventToolResult, Data: ev.Data}, true

	case orchestration.EventFinal:
		return database.StreamEvent{Type: database.EventText, Content: ev.Content}, true

	case orchestration.EventError:
		return database.StreamEvent{Type: database.EventError, Data: ev.Data}, true

	case orchestration.EventDone:
		return database.StreamEvent{Type: database.EventDone}, true
	}
	return database.StreamEvent{}, false
}

// The orchestrator has already logged the real cause; shipping it to the client
// put "agent market_data: produced no output" on screen.
func stepFailureMessage(err string) string {
	if strings.Contains(err, context.DeadlineExceeded.Error()) {
		return "This step ran out of time and was skipped."
	}
	// In max mode a spent quota fails per step, so the chips are where it shows.
	if isQuotaFailure(err) {
		return "This step was skipped: the AI provider quota has run out."
	}
	return "This step did not return a usable result and was skipped."
}

func renderPlan(plan orchestration.Plan) string {
	var sb strings.Builder
	sb.WriteString("Plan:\n")
	for i, step := range plan.Steps {
		fmt.Fprintf(&sb, "%d. [%s] %s\n", i+1, step.Agent, step.Task)
	}
	// A blank line, not just the item's newline: the client appends the step prose
	// onto this same block, and one newline does not close a markdown list.
	sb.WriteString("\n")
	return sb.String()
}

func stepArguments(info orchestration.StepInfo) string {
	args, err := json.Marshal(map[string]any{
		"step":   fmt.Sprintf("%d/%d", info.Index, info.Total),
		"task":   info.Task,
		"reason": info.Reason,
	})
	if err != nil {
		return ""
	}
	return string(args)
}

// history ends with the message being answered: that message alone is the goal,
// everything before it becomes planning context. Kept apart deliberately — the
// planner needs the conversation to resolve a follow-up ("and what about VNM?"),
// while the agents must see the request as typed, because that is where they take
// the answer's language from.
func pipelineRun(history []database.Message, summary database.ConversationSummary) orchestration.Request {
	if len(history) == 0 {
		return orchestration.Request{}
	}
	req := orchestration.Request{Goal: history[len(history)-1].Content}

	prior := history[:len(history)-1]
	if len(prior) > pipelineGoalTurns {
		prior = prior[len(prior)-pipelineGoalTurns:]
	}
	if len(prior) == 0 && summary.Summary == "" && len(summary.KeyFacts) == 0 {
		return req
	}

	var sb strings.Builder
	if summary.Summary != "" {
		fmt.Fprintf(&sb, "CONVERSATION SO FAR:\n%s\n\n", summary.Summary)
	}
	if len(summary.KeyFacts) > 0 {
		fmt.Fprintf(&sb, "KEY FACTS:\n- %s\n\n", strings.Join(summary.KeyFacts, "\n- "))
	}
	if len(prior) > 0 {
		sb.WriteString("RECENT MESSAGES:\n")
		for _, m := range prior {
			// Cut by rune, not byte: the conversation is routinely Vietnamese and a
			// byte cut would split a character in half.
			content := strings.TrimSpace(m.Content)
			if runes := []rune(content); len(runes) > pipelineGoalChars {
				content = string(runes[:pipelineGoalChars]) + "…"
			}
			fmt.Fprintf(&sb, "%s: %s\n", m.Role, content)
		}
	}
	req.History = strings.TrimRight(sb.String(), "\n")
	return req
}
