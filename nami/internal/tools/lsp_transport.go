package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func (c *lspClient) request(ctx context.Context, method string, params any, result any) error {
	id := c.nextRequestID()
	message := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	if err := c.writeMessage(message); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		envelope, err := c.readMessage()
		if err != nil {
			stderr := strings.TrimSpace(c.stderr.String())
			if stderr != "" {
				return fmt.Errorf("read lsp response for %s: %w (%s)", method, err, stderr)
			}
			return fmt.Errorf("read lsp response for %s: %w", method, err)
		}
		if envelope.Method != "" {
			continue
		}
		responseID, ok := parseLSPResponseID(envelope.ID)
		if !ok || responseID != id {
			continue
		}
		if envelope.Error != nil {
			return fmt.Errorf("lsp %s failed: %s", method, envelope.Error.Message)
		}
		if result == nil || len(envelope.Result) == 0 || string(envelope.Result) == "null" {
			return nil
		}
		if err := json.Unmarshal(envelope.Result, result); err != nil {
			return fmt.Errorf("decode lsp result for %s: %w", method, err)
		}
		return nil
	}
}

func (c *lspClient) notify(method string, params any) error {
	message := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return c.writeMessage(message)
}

func (c *lspClient) writeMessage(message any) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode lsp message: %w", err)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if _, err := fmt.Fprintf(c.stdin, "Content-Length: %d\r\n\r\n", len(data)); err != nil {
		return err
	}
	if _, err := c.stdin.Write(data); err != nil {
		return err
	}
	return nil
}

func (c *lspClient) readMessage() (lspResponseEnvelope, error) {
	contentLength := 0
	for {
		line, err := c.stdout.ReadString('\n')
		if err != nil {
			return lspResponseEnvelope{}, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(parts[0]), "Content-Length") {
			value := strings.TrimSpace(parts[1])
			length, err := strconv.Atoi(value)
			if err != nil {
				return lspResponseEnvelope{}, fmt.Errorf("parse content length %q: %w", value, err)
			}
			contentLength = length
		}
	}
	if contentLength <= 0 {
		return lspResponseEnvelope{}, fmt.Errorf("missing content length in lsp message")
	}
	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(c.stdout, payload); err != nil {
		return lspResponseEnvelope{}, err
	}
	var envelope lspResponseEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return lspResponseEnvelope{}, fmt.Errorf("decode lsp message: %w", err)
	}
	return envelope, nil
}

func (c *lspClient) nextRequestID() int64 {
	c.nextID++
	return c.nextID
}

func (c *lspClient) Close() error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = c.request(shutdownCtx, "shutdown", map[string]any{}, nil)
	_ = c.notify("exit", map[string]any{})
	_ = c.stdin.Close()
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	_ = c.cmd.Wait()
	return nil
}

func pathToFileURI(path string) string {
	resolved := filepath.Clean(path)
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(resolved)}).String()
}

func fileURIToPath(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "file" {
		return value
	}
	path := filepath.FromSlash(parsed.Path)
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '\\' && path[2] == ':' {
		path = path[1:]
	}
	return filepath.Clean(path)
}

func parseLSPResponseID(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var numeric int64
	if err := json.Unmarshal(raw, &numeric); err == nil {
		return numeric, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		parsed, err := strconv.ParseInt(text, 10, 64)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}
