package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// VCIHTTPTimeout bounds a single VietCap REST call. The symbol directory is the
// one payload large enough to need its own, longer budget.
const VCIHTTPTimeout = 10 * time.Second

// ErrNoIQData reports that iq-insight answered successfully but carried no
// payload, which is how it signals an unknown ticker. Callers use it to tell
// "no such symbol" apart from "the endpoint is broken".
var ErrNoIQData = errors.New("iq-insight returned no data")

// iqBaseURL is IQ_INSIGHT_URL indirected through a var so tests can point it at
// a stub server. Nothing outside this package writes it.
var iqBaseURL = IQ_INSIGHT_URL

// iqEnvelope is the wrapper every iq-insight-service response arrives in. Data
// stays raw so an absent payload can be told apart from a decoded zero value.
type iqEnvelope struct {
	Successful bool            `json:"successful"`
	Status     int             `json:"status"`
	Msg        string          `json:"msg"`
	TraceID    string          `json:"traceId"`
	Data       json.RawMessage `json:"data"`
}

// FetchIQInsight performs a GET against VietCap's iq-insight-service,
// decompresses the response, and unwraps the standard envelope into T.
//
// Three failure surfaces are checked, because no single one of them covers the
// others (all verified live against VietCap):
//
//	unknown ticker    HTTP 200, successful=true,  status=200, data=null
//	bad query param   HTTP 200, successful=false, status=400, data=null
//	missing endpoint  HTTP 404, successful=false, status=404, data=null
//
// A decommissioned endpoint that answers HTTP 200 with a bare `{}` decodes to a
// zero envelope and is caught by the status check. This is what turns a dead
// endpoint into a loud failure instead of an empty result.
func FetchIQInsight[T any](ctx context.Context, path string) (T, error) {
	var payload T

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, iqBaseURL+path, nil)
	if err != nil {
		return payload, fmt.Errorf("create iq-insight request for %s: %w", path, err)
	}
	for k, v := range VCI_HEADERS {
		req.Header.Set(k, v)
	}

	resp, err := (&http.Client{Timeout: VCIHTTPTimeout}).Do(req)
	if err != nil {
		return payload, fmt.Errorf("fetch %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return payload, fmt.Errorf("fetch %s: unexpected status %d", path, resp.StatusCode)
	}

	reader, err := GZIPCompression(resp.Body, resp.Header.Get("Content-Encoding"))
	if err != nil {
		return payload, fmt.Errorf("decompress %s: %w", path, err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		return payload, fmt.Errorf("read %s: %w", path, err)
	}

	var env iqEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return payload, fmt.Errorf("unmarshal iq-insight envelope for %s: %w", path, err)
	}

	// Reject an envelope the service itself flagged, or one it never filled in
	if !env.Successful || env.Status != http.StatusOK {
		return payload, fmt.Errorf("iq-insight rejected %s: status %d, msg %q, trace %q",
			path, env.Status, env.Msg, env.TraceID)
	}

	// Reject an absent payload before decoding it into a misleading zero value
	if len(env.Data) == 0 || bytes.Equal(bytes.TrimSpace(env.Data), []byte("null")) {
		return payload, fmt.Errorf("%w from %s", ErrNoIQData, path)
	}

	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return payload, fmt.Errorf("unmarshal iq-insight data for %s: %w", path, err)
	}

	return payload, nil
}
