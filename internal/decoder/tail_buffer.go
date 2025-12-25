package decoder

import "sync"

type tailBuffer struct {
	mu           sync.Mutex
	maxLines     int
	maxLineBytes int
	lines        []string
}

func newTailBuffer(maxLines int, maxLineBytes int) *tailBuffer {
	if maxLines < 0 {
		maxLines = 0
	}
	if maxLineBytes <= 0 {
		maxLineBytes = 16 * 1024
	}
	return &tailBuffer{maxLines: maxLines, maxLineBytes: maxLineBytes, lines: make([]string, 0, maxLines)}
}

func (t *tailBuffer) add(line string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.maxLines == 0 {
		return
	}
	if len(line) > t.maxLineBytes {
		line = line[:t.maxLineBytes]
	}
	if len(t.lines) < t.maxLines {
		t.lines = append(t.lines, line)
		return
	}
	copy(t.lines, t.lines[1:])
	t.lines[len(t.lines)-1] = line
}

func (t *tailBuffer) snapshot() []string {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]string, 0, len(t.lines))
	out = append(out, t.lines...)
	return out
}
