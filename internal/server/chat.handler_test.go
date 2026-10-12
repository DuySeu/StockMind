package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseChatInputReadsTraceFlag(t *testing.T) {
	t.Parallel()

	// Build the same request as JSON and as form-data
	jsonReq := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"content":"FPT?","trace":true}`))
	jsonReq.Header.Set("Content-Type", "application/json")

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	_ = form.WriteField("content", "FPT?")
	_ = form.WriteField("trace", "true")
	_ = form.Close()
	formReq := httptest.NewRequest(http.MethodPost, "/v1/chat", &body)
	formReq.Header.Set("Content-Type", form.FormDataContentType())

	untraced := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"content":"FPT?"}`))
	untraced.Header.Set("Content-Type", "application/json")

	tests := []struct {
		name string
		req  *http.Request
		want bool
	}{
		{"reads trace from json body", jsonReq, true},
		{"reads trace from form data", formReq, true},
		{"defaults trace to false", untraced, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in, ok := parseChatInput(httptest.NewRecorder(), tt.req)
			if !ok || in.Trace != tt.want {
				t.Errorf("parseChatInput() = (Trace %v, ok %v); want (Trace %v, ok true)", in.Trace, ok, tt.want)
			}
		})
	}
}
