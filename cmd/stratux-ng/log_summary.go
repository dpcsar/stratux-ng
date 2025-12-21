package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"stratux-ng/internal/replay"
)

type logSummary struct {
	Segments    int
	Frames      int
	Invalid     int
	MaxDuration time.Duration
	MsgIDCounts map[byte]int
}

func summarizeGDL90Log(records []replay.Record) logSummary {
	s := logSummary{MsgIDCounts: map[byte]int{}}
	if len(records) == 0 {
		return s
	}

	origin := time.Duration(0)
	hasFrames := false
	segments := 0

	for _, r := range records {
		if r.Frame == nil {
			segments++
			origin = r.At
			continue
		}
		hasFrames = true

		s.Frames++
		at := r.At - origin
		if at < 0 {
			at = 0
		}
		if at > s.MaxDuration {
			s.MaxDuration = at
		}

		msgID, ok := msgIDFromFramedGDL90(r.Frame)
		if !ok {
			s.Invalid++
			continue
		}
		s.MsgIDCounts[msgID]++
	}
	if segments == 0 && hasFrames {
		segments = 1
	}
	s.Segments = segments

	return s
}

// msgIDFromFramedGDL90 extracts the message ID from a framed+escaped GDL90 packet.
// It intentionally does not verify CRC (summary tool is best-effort).
func msgIDFromFramedGDL90(frame []byte) (byte, bool) {
	if len(frame) < 4 {
		return 0, false
	}
	if frame[0] != 0x7E || frame[len(frame)-1] != 0x7E {
		return 0, false
	}

	// De-escape and strip flags.
	raw := make([]byte, 0, len(frame))
	for i := 1; i < len(frame)-1; i++ {
		b := frame[i]
		if b == 0x7D {
			i++
			if i >= len(frame)-1 {
				return 0, false
			}
			raw = append(raw, frame[i]^0x20)
			continue
		}
		raw = append(raw, b)
	}
	if len(raw) < 3 {
		return 0, false
	}

	msg := raw[:len(raw)-2] // strip CRC16
	if len(msg) == 0 {
		return 0, false
	}
	return msg[0], true
}

func printLogSummary(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	recs, err := replay.NewReader(f).ReadAll()
	if err != nil {
		return err
	}

	s := summarizeGDL90Log(recs)

	fmt.Printf("path: %s\n", path)
	fmt.Printf("segments: %d\n", s.Segments)
	fmt.Printf("frames: %d\n", s.Frames)
	fmt.Printf("invalid_frames: %d\n", s.Invalid)
	fmt.Printf("max_duration: %s\n", s.MaxDuration)

	keys := make([]int, 0, len(s.MsgIDCounts))
	for k := range s.MsgIDCounts {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	fmt.Printf("msg_id_counts:\n")
	for _, k := range keys {
		b := byte(k)
		fmt.Printf("  0x%02X: %d\n", b, s.MsgIDCounts[b])
	}
	return nil
}
