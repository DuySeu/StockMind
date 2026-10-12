# Eval trace design

Một trang `/eval` riêng để chạy một lượt chat qua đúng `POST /v1/chat` và xem trace của từng bước bên trong: guardrail, orchestrator, từng agent, từng vòng gọi LLM, function calling, MCP. Mỗi bước hiện tên, thứ tự, cha-con, input/output, thời gian chạy, và tổng thời gian của cả lượt. Luồng này tách khỏi logic chính: production không bật trace thì không đổi hành vi và không tốn thêm gì.

## Mục tiêu và phạm vi

Trong phạm vi:

- Trace một lượt chat (single-agent hoặc max mode), hiển thị trực tiếp khi span đến.
- Lượt chạy đi qua đúng đường code thật, kể cả persist message và summarize. Conversation tạo từ trang eval nằm chung DB với chat thường.
- Thêm bước mới vào trace sau này chỉ cần một dòng `trace.Start` ở entry point của bước đó, UI không phải sửa.

Ngoài phạm vi:

- Lưu trace hoặc so sánh giữa các lần chạy. Trace chỉ sống trong SSE stream và state của trình duyệt, reload là mất.
- Bộ test case có assertion, chạy batch, chấm pass/fail.
- OpenTelemetry hoặc exporter ra Jaeger/Langfuse. Span có cùng shape cơ bản (name, parent, start, duration, attrs) nên có thể chuyển sang sau.

## Quyết định đã chốt

| Quyết định | Lựa chọn | Lý do |
|---|---|---|
| Cách thu trace | Span recorder tự viết, truyền qua `context.Context` | Thấy được bước nội bộ (redact, classify, planner, MCP reconnect) mà SSE hiện tại không có; không thêm dependency |
| Cách bật | Field `trace: true` trên `/v1/chat`, không có biến môi trường nào chặn | Dùng đúng API của app, không cần cấu hình để chạy `/eval`. Đánh đổi: app chưa có auth, nên mọi client đều có thể xin trace và thấy system prompt cùng output tool thô |
| Cách gửi span | SSE event `trace` trên cùng stream của lượt chat | Không cần endpoint hay storage mới |
| Lưu trace | Không | Phạm vi bản đầu |
| Ghi DB khi chạy từ eval | Ghi như chat thật | Trace phản ánh đúng production; cho phép test câu nối tiếp bằng `session_id` |
| Recorder đầy | Bỏ span, đếm số bị bỏ, không block | Trace không được làm chậm hay treo lượt chat |

## Kiến trúc

Package mới `internal/trace`, không import package nội bộ nào khác, nên mọi package (`guardrail`, `llm`, `orchestration`, `agents`, `tools`, `mcp`, `server`) đều có thể gọi nó.

```go
type Span struct {
	ID         string         `json:"id"`
	ParentID   string         `json:"parent_id,omitempty"`
	Kind       string         `json:"kind"` // handler | guardrail | orchestrator | agent | llm | tool | mcp
	Name       string         `json:"name"` // "guardrail.classify", "tool.get_report", "llm.round#2"
	Start      time.Time      `json:"start"`
	DurationMs int64          `json:"duration_ms"`
	Input      any            `json:"input,omitempty"`
	Output     any            `json:"output,omitempty"`
	Attrs      map[string]any `json:"attrs,omitempty"`
	Error      string         `json:"error,omitempty"`
	Truncated  bool           `json:"truncated,omitempty"`
}

ctx, end := trace.Start(ctx, "tool", "tool.get_report", args)
defer end(result, err)

trace.Annotate(ctx, "category", decision.Category)
```

| Thành phần | Nhiệm vụ |
|---|---|
| `Recorder` | Giữ buffered channel các event `span_start`/`span_end` và bộ đếm `dropped`. Send không block: channel đầy thì tăng `dropped` |
| `WithRecorder(ctx, rec)` | Gắn recorder vào context |
| `Start(ctx, kind, name, input)` | Tạo span con của span hiện tại trong `ctx`, gửi `span_start`, trả về ctx mới và hàm `end(output, err)`. Không có recorder trong `ctx` thì trả về ctx cũ và `end` rỗng |
| `Annotate(ctx, key, value)` | Thêm attr vào span hiện tại; no-op khi không có recorder |

`Input` được serialize và cắt ở 8 KB khi `Start`, `Output` được cắt ở 8 KB khi gọi `end`, kèm `Truncated: true`. Span phải an toàn khi gọi từ nhiều goroutine, vì `LLMChat` và `Orchestrator.Run` chạy trong goroutine riêng: `Annotate` và `end` sửa span dưới mutex của span, và mỗi event gửi vào channel là một bản copy giá trị của `Span` (kể cả map `Attrs`), không phải con trỏ. Event `span_start` chỉ có `Input` và attrs có lúc đó; bản đầy đủ là event `span_end`, và UI lấy bản `span_end` khi đã có.

### Bật trace trên `/v1/chat`

`parseChatInput` đọc thêm `trace` vào `chatInput.Trace` (JSON body và form-data, giống `max_mode`). SSE dùng hằng mới `database.EventTrace StreamEventType = "trace"` trong `internal/database/types.go`, cạnh `EventGuardrail`. Khi `chatInput.Trace` bật, `ChatHandler`:

1. Tạo `Recorder`, gắn vào `r.Context()`, mở span gốc `chat.turn` với attr `max_mode`. `session_id` được thêm bằng `Annotate` sau khi đã có (session mới chỉ có ID sau `CreateConversation`).
2. Sau khi gửi SSE `start`, xả các span đã buffer (redact, history load).
3. Vòng `relay:` có thêm một `case` đọc channel của recorder và ghi SSE `trace`. Mọi lần ghi vào `w` vẫn nằm trên một goroutine, nên không có race.
4. Hiện nhánh `EventDone` trong vòng relay ghi `done` ngay (`chat.handler.go`, `case database.EventDone`), trong khi vòng lặp còn chạy tới khi channel đóng, nên span cuối sẽ đến sau `done`. Khi có recorder, nhánh đó chỉ đặt `sawDone` mà không ghi; sau vòng lặp, nếu `!clientGone`, handler đóng span `chat.turn`, xả phần còn lại, gửi event `summary` có `dropped`, rồi mới ghi `done`, kể cả khi `sawDone` đã là true (điều kiện `!sawDone` hiện tại chỉ áp khi không có recorder). Không có recorder thì giữ nguyên hành vi hiện tại.
5. Mọi đường kết thúc sớm sau khi đã gửi header SSE đều đi qua `endTurn` (lỗi load history, lỗi render prompt, lỗi lưu tin user, lỗi `turnStream`, nhánh guardrail chặn lượt). `endTurn` nhận thêm recorder (có thể `nil`) và xả trace cùng `summary` trước khi ghi `done`. Nhánh classify lỗi do client đã đóng kết nối không ghi gì thêm, như hiện tại.

Span kết thúc sau khi client đóng kết nối thì không còn ai đọc: recorder bị bỏ đi cùng request, và UI giữ các span đó ở trạng thái `running`.

Payload của SSE `trace`: `{"phase":"start"|"end","span":Span}`, và event cuối `{"phase":"summary","dropped":N}`.

### Điểm gắn span

| Span | Vị trí | Attrs chính | Output |
|---|---|---|---|
| `chat.turn` (gốc) | `ChatHandler` | `session_id`, `max_mode` | câu trả lời cuối và lỗi của lượt; nhánh bị chặn ghi câu từ chối và `guardrail: blocked` |
| `guardrail.redact` | ngay sau `parseChatInput` | | bản đã mask và `kinds`. Không ghi input, để giữ quy tắc của guardrail spec là bản gốc chưa mask không xuất hiện ở đâu cả |
| `history.load` | `ChatHandler` | số message | không có |
| `guardrail.classify` | `Classifier.Classify` | `fail_open` | `Decision{Allowed, Category, Reason}` |
| `persist.<role>` | `persistMessageCtx` | | không có. Chỉ các lần ghi đồng bộ có span (tin user, câu từ chối) |
| `llm.round#n` | vòng lặp trong `LLMChat` | `text_len`, `thinking_len`, `tool_calls`, `ttft_ms` | text của vòng và các tool call model yêu cầu |
| `tool.<name>` | `tools.Manager.Execute` | | kết quả tool trước khi sanitize |
| `guardrail.sanitize` | `runToolRound`, quanh lời gọi `s.sanitize` | `source`, `changed`, `len_before`, `len_after`. Chữ ký `func(source, result string) string` giữ nguyên | kết quả sau khi sanitize |
| `mcp.call` | `mcp.Manager.CallTool` | `reconnected` | kết quả từ MCP server |
| `orchestrator.plan` | `Orchestrator.Run` | `steps` | plan của planner |
| `planner.attempt#n` | `Planner.Plan` | | plan của lần thử đó, lỗi validate ở `error` |
| `orchestrator.step` | `Orchestrator.runStep`, mở trước `context.WithTimeout` để `agent.Run` nhận ctx con của span | `timeout` | nội dung cuối của step, kể cả bản dở dang khi step lỗi |
| `agent.<name>` | `LLMAgent.Run` | `tools` | nội dung cuối của agent |

Model của provider chưa được ghi vào `llm.round#n`, vì `models.Provider` không lộ tên model.

Vì cây cha-con lấy từ `ctx`, trong max mode cây là `chat.turn` > `orchestrator.step` > `agent.market_data` > (`llm.round#1`, `tool.get_stock_price`, `llm.round#2`). Span `llm.round#n` đóng khi provider gửi `done` cho vòng đó, nên `tool.*` là anh em của `llm.round#n` (cùng cha), chạy sau vòng đã yêu cầu nó; `runToolRound` vẫn nhận ctx của `LLMChat`, không nhận ctx của vòng.

Không trace: ghi tin assistant và `maybeSummarize`. Hai việc này chạy trong goroutine với `context.Background()` sau khi đã gửi `done`, nên không thuộc trace của lượt. Các span có điều kiện: `guardrail.redact` và `guardrail.classify` chỉ có khi guardrail bật, `history.load` chỉ có khi tiếp session cũ.

`ttft_ms` là thời gian từ lúc gọi `provider.Completion` tới delta `text` hoặc `thinking` đầu tiên của vòng đó.

## Chạy từng module

Ngoài full chat, trang `/eval` chạy được từng module một cách độc lập qua nhóm route `/v1/eval` (`internal/server/eval.handler.go`). Mỗi endpoint gọi đúng hàm Go mà luồng chat dùng, không đi qua `ChatHandler`, không ghi DB và không chạy guardrail trừ khi chính module đó là guardrail. Tất cả stream cùng một dạng: event `trace` trong lúc chạy, rồi `result` (hoặc `error` với message gốc, vì trang eval cần thấy lỗi thật), rồi `done`. Span gốc là `eval.<module>`.

| Module | Endpoint | Input | Code chạy | Dừng ở |
|---|---|---|---|---|
| Full chat | `POST /v1/chat` + `trace` | câu hỏi, max_mode, session | toàn bộ lượt | câu trả lời |
| Guardrail | `POST /v1/eval/guardrail` | `message`, `history` tùy chọn | `Redact` rồi `Classifier.Classify` | `Decision`; lỗi fail-open nằm trong `fail_open` của result |
| Sanitize | `POST /v1/eval/sanitize` | `source`, `content` | `guardrail.Sanitize` | text đã làm sạch, `changed`, bản bọc `WrapToolOutput` |
| Planner | `POST /v1/eval/plan` | `goal` | `Planner.Plan` | `Plan`, không chạy agent |
| Pipeline | `POST /v1/eval/pipeline` | `goal` | `Orchestrator.Collect` | `Result` (plan, các step, final) |
| Agent | `POST /v1/eval/agent` | `agent`, `task`, `goal` tùy chọn, `context` tùy chọn | `LLMAgent.Run` | `Output` của agent |
| Tool / MCP | `POST /v1/eval/tool` | `tool`, `args` JSON | `tools.Manager.Execute`, gồm cả tool MCP bridge | kết quả tool |

`GET /v1/eval/catalog` trả danh sách agent (`Registry.Roster`), tool kèm schema (`tools.Manager.All`) và `guardrailEnabled`, để form trên FE có dropdown. `ServerDeps` nhận thêm `Planner`, `Registry`, `Tools` từ `cmd/main.go`. Khi `GUARDRAIL_ENABLED=false`, module guardrail chỉ chạy redact và ghi chú trong result.

## Frontend

Route `/eval` trong `frontend/src/router.tsx`, nằm ngoài `MainLayout` như `/c`. Không thêm dependency.

| Thành phần | File | Nhiệm vụ |
|---|---|---|
| `EvalPage` | `src/pages/EvalPage.tsx` | Bộ chọn module, gọi `sendChatMessage` với `trace: true` cho full chat hoặc `runEvalModule` cho module riêng, giữ danh sách event trong state, hiện câu trả lời (markdown) hoặc result (JSON) |
| `ModuleForm` | `src/components/eval/ModuleForm.tsx` | Form của từng module, giữ giá trị khi đổi module, validate trước khi gửi |
| `JsonPayload` | `src/components/eval/JsonPayload.tsx` | Khối JSON có nút copy, dùng cho input/output của span và result |
| API eval | `src/api/eval.ts` | `getEvalCatalog`, `runEvalModule`; stream dùng chung `postSSE` trong `src/api/chat.ts` |
| `buildSpanTree` | `src/lib/trace.ts` | Hàm thuần: danh sách event `trace` thành cây, sắp theo `start`. Span có `parent_id` chưa tới thì tạm đặt dưới gốc; span chưa có `end` được đánh dấu `running` |
| `SpanWaterfall` | `src/components/eval/SpanWaterfall.tsx` | Mỗi dòng một span, thụt lề theo độ sâu, collapse được, thanh thời gian trên trục chung của cả lượt, màu theo `kind`, đỏ khi có `error` |
| `SpanDetail` | `src/components/eval/SpanDetail.tsx` | Panel bên phải khi chọn một span: kind, duration, status, attrs, input/output dạng JSON có nút copy và dấu `truncated` |
| Thanh tổng | trong `EvalPage` | Tổng thời gian, số span, kết quả guardrail cuối cùng (allowed/blocked, category), số span bị bỏ |

`src/api/chat.ts`: thêm `"trace"` vào `ChatEventType`, và thêm tham số cuối tùy chọn `options?: { trace?: boolean }` cho `sendChatMessage` (sau `signal`), nên caller hiện có trong `Chatbot.tsx` không phải sửa.

```
+--------------------------------------------------------------------------+
| Question: [FPT co nen mua khong?            ] [x] max_mode [Run] session  |
| Total 18.4s | 23 spans | guardrail: allowed (stock) | 0 dropped           |
+-----------------------------------------------+--------------------------+
| span                         ms   0s      10s 18s| guardrail.classify      |
| v chat.turn               18412 [=================]| kind: guardrail 612 ms |
|   guardrail.redact            1 |                 | attrs / input / output |
|   guardrail.classify        612 [=]               |                        |
|   orchestrator.plan        2210  [==]             |                        |
| v orchestrator.step        4125     [====]        |                        |
|   v agent.market_data      4120     [====]        |                        |
|       llm.round#1          1800     [==]          |                        |
|       tool.get_stock_price  410       [.]         |                        |
|       llm.round#2          1900       [==]        |                        |
+-----------------------------------------------+--------------------------+
| Final answer (markdown)                                                   |
+--------------------------------------------------------------------------+
```

## Xử lý lỗi

| Tình huống | Xử lý |
|---|---|
| Channel của recorder đầy | Bỏ span, tăng `dropped`, gửi trong event `summary` |
| Client đóng kết nối giữa lượt | Như nhánh `clientGone` hiện có; span chưa đóng ở lại trạng thái `running` trên UI |
| Span kết thúc với lỗi (tool lỗi, classify fail-open) | Ghi `Error`, thanh màu đỏ; lượt chat không bị ảnh hưởng |
| Payload lớn (system prompt, kết quả RAG) | Cắt 8 KB mỗi field (`Input` khi `Start`, `Output` khi `end`) |

Chi phí: mỗi lượt có trace gửi thêm khoảng 20-60 event SSE, mỗi event tối đa vài chục KB. Lượt không bật trace không đổi.

## Test

Theo kiểu table test và tên test mô tả case, như các package hiện có.

- `internal/trace/trace_test.go`: `start_without_recorder_is_noop`, `child_span_gets_parent_id_from_ctx`, `drops_span_when_buffer_full_without_blocking`, `truncates_large_output`, `annotate_adds_attr_to_current_span`.
- `internal/trace/trace_test.go` thêm `span_start_event_is_not_mutated_by_later_annotate`, chạy với `-race`.
- `internal/llm/service_test.go`: với fake provider có một tool call, recorder nhận `llm.round#1`, `tool.<name>`, `llm.round#2` theo đúng thứ tự, cả ba cùng một cha.
- `internal/server/eval.handler_test.go`: `eval_sanitize_streams_trace_then_result_then_done`, `eval_tool_nests_tool_span_under_eval_root`, `eval_rejects_missing_field`.
- `internal/server/chat.handler_test.go` (file mới): chỉ test phần thuần, không dựng DB. `parse_chat_input_reads_trace_flag` cho JSON, form-data và trường hợp không gửi cờ.
- Frontend chưa có test runner và không thêm trong spec này; `buildSpanTree` được kiểm tra tay trên trang `/eval`.
