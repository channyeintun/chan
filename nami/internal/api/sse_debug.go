package api

import (
	"io"
	"time"

	"github.com/channyeintun/nami/internal/debuglog"
)

type providerStreamDebugReader struct {
	provider  string
	reader    io.Reader
	startedAt time.Time
	seenBytes bool
	closed    bool
}

// sseBodyWithDebug wraps an io.Reader with debug logging when enabled.
func sseBodyWithDebug(body io.Reader, provider string) io.Reader {
	debuglog.Log("provider_stream", "stream_opened", map[string]any{
		"provider": provider,
	})
	return &providerStreamDebugReader{
		provider:  provider,
		reader:    debuglog.NewSSEReaderProxy(body, provider),
		startedAt: time.Now(),
	}
}

func (r *providerStreamDebugReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && !r.seenBytes {
		r.seenBytes = true
		debuglog.Log("provider_stream", "stream_first_bytes", map[string]any{
			"provider":    r.provider,
			"elapsed_ms":  time.Since(r.startedAt).Milliseconds(),
			"first_bytes": n,
		})
	}
	if err != nil && !r.closed {
		r.closed = true
		fields := map[string]any{
			"provider":    r.provider,
			"elapsed_ms":  time.Since(r.startedAt).Milliseconds(),
			"saw_bytes":   r.seenBytes,
			"error_kind":  "stream_read",
			"end_of_file": err == io.EOF,
		}
		if err != io.EOF {
			fields["error"] = err.Error()
		}
		debuglog.Log("provider_stream", "stream_closed", fields)
	}
	return n, err
}
