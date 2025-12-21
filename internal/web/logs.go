package web

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LogBuffer struct {
	mu      sync.Mutex
	max     int
	lines   []string
	partial string
	dropped uint64
}

func NewLogBuffer(maxLines int) *LogBuffer {
	if maxLines <= 0 {
		maxLines = 2000
	}
	return &LogBuffer{max: maxLines}
}

// Write implements io.Writer. It collects logs as lines.
func (b *LogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Combine any previous partial line with this chunk.
	data := append([]byte(b.partial), p...)
	b.partial = ""

	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Allow long lines.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		b.appendLineLocked(scanner.Text())
	}
	// If last byte is not a newline, scanner will have returned the last token;
	// we can't tell if it was partial. Use a conservative heuristic: if data
	// doesn't end with '\n', treat the last scanned line as partial and remove it.
	if len(data) > 0 && data[len(data)-1] != '\n' {
		if len(b.lines) > 0 {
			b.partial = b.lines[len(b.lines)-1]
			b.lines = b.lines[:len(b.lines)-1]
		}
	}

	return len(p), nil
}

func (b *LogBuffer) appendLineLocked(line string) {
	line = strings.TrimRight(line, "\r")
	if line == "" {
		return
	}
	b.lines = append(b.lines, line)
	if len(b.lines) > b.max {
		over := len(b.lines) - b.max
		b.lines = b.lines[over:]
		b.dropped += uint64(over)
	}
}

type LogsResponse struct {
	NowUTC  string   `json:"now_utc"`
	Dropped uint64   `json:"dropped"`
	Lines   []string `json:"lines"`
}

func (b *LogBuffer) Snapshot(tail int) (lines []string, dropped uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	dropped = b.dropped
	if tail <= 0 {
		tail = 200
	}
	if tail > len(b.lines) {
		tail = len(b.lines)
	}
	start := len(b.lines) - tail
	lines = append([]string(nil), b.lines[start:]...)
	return lines, dropped
}

func (b *LogBuffer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		tail := 200
		if s := strings.TrimSpace(r.URL.Query().Get("tail")); s != "" {
			v, err := strconv.Atoi(s)
			if err != nil || v < 1 || v > 5000 {
				http.Error(w, "tail must be an integer in [1,5000]", http.StatusBadRequest)
				return
			}
			tail = v
		}

		lines, dropped := b.Snapshot(tail)
		resp := LogsResponse{
			NowUTC:  time.Now().UTC().Format(time.RFC3339Nano),
			Dropped: dropped,
			Lines:   lines,
		}

		if strings.EqualFold(r.URL.Query().Get("format"), "text") {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			if dropped > 0 {
				_, _ = fmt.Fprintf(w, "[dropped=%d]\n", dropped)
			}
			for _, line := range lines {
				_, _ = w.Write([]byte(line))
				_, _ = w.Write([]byte("\n"))
			}
			return
		}

		bts, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			http.Error(w, "marshal failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(bts)
		_, _ = w.Write([]byte("\n"))
	})
}
