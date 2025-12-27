package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"stratux-ng/internal/config"
)

type SettingsPayload struct {
	GDL90Dest       string `json:"gdl90_dest"`
	Interval        string `json:"interval"`
	OwnshipICAO     string `json:"ownship_icao"`
	OwnshipCallsign string `json:"ownship_callsign"`
}

// SettingsPayloadIn is the strict POST schema.
//
// All fields are required (no partial updates) to avoid hidden defaults and
// prevent accidental schema drift.
type SettingsPayloadIn struct {
	GDL90Dest       *string `json:"gdl90_dest"`
	Interval        *string `json:"interval"`
	OwnshipICAO     *string `json:"ownship_icao"`
	OwnshipCallsign *string `json:"ownship_callsign"`
}

var settingsPostKeys = []string{
	"gdl90_dest",
	"interval",
	"ownship_icao",
	"ownship_callsign",
}

func decodeSettingsPayloadInStrict(body []byte) (SettingsPayloadIn, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()

	// First pass: stream tokens to enforce strict object rules and detect duplicate keys.
	allowed := make(map[string]struct{}, len(settingsPostKeys))
	for _, k := range settingsPostKeys {
		allowed[k] = struct{}{}
	}
	seen := make(map[string]struct{}, len(settingsPostKeys))

	tok, err := dec.Token()
	if err != nil {
		return SettingsPayloadIn{}, fmt.Errorf("invalid json: %w", err)
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return SettingsPayloadIn{}, errors.New("invalid json: expected object")
	}

	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return SettingsPayloadIn{}, fmt.Errorf("invalid json: %w", err)
		}
		key, ok := kt.(string)
		if !ok {
			return SettingsPayloadIn{}, errors.New("invalid json: expected string key")
		}
		if _, ok := allowed[key]; !ok {
			return SettingsPayloadIn{}, fmt.Errorf("invalid json: unknown key %q", key)
		}
		if _, dup := seen[key]; dup {
			return SettingsPayloadIn{}, fmt.Errorf("invalid json: duplicate key %q", key)
		}
		seen[key] = struct{}{}

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return SettingsPayloadIn{}, fmt.Errorf("invalid json: %w", err)
		}
		if strings.TrimSpace(string(raw)) == "null" {
			return SettingsPayloadIn{}, fmt.Errorf("invalid json: %q cannot be null", key)
		}
	}

	end, err := dec.Token()
	if err != nil {
		return SettingsPayloadIn{}, fmt.Errorf("invalid json: %w", err)
	}
	delim, ok = end.(json.Delim)
	if !ok || delim != '}' {
		return SettingsPayloadIn{}, errors.New("invalid json: expected end of object")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return SettingsPayloadIn{}, errors.New("invalid json: trailing data")
	}

	for _, k := range settingsPostKeys {
		if _, ok := seen[k]; !ok {
			return SettingsPayloadIn{}, fmt.Errorf("invalid json: missing required key %q", k)
		}
	}

	// Second pass: decode into the typed struct.
	var out SettingsPayloadIn
	dec2 := json.NewDecoder(bytes.NewReader(body))
	dec2.DisallowUnknownFields()
	if err := dec2.Decode(&out); err != nil {
		return SettingsPayloadIn{}, fmt.Errorf("invalid json: %w", err)
	}
	if err := dec2.Decode(&struct{}{}); err != io.EOF {
		return SettingsPayloadIn{}, errors.New("invalid json: trailing data")
	}

	return out, nil
}

func configToSettingsPayload(cfg config.Config) SettingsPayload {
	return SettingsPayload{
		GDL90Dest:       cfg.GDL90.Dest,
		Interval:        cfg.GDL90.Interval.String(),
		OwnshipICAO:     cfg.Ownship.ICAO,
		OwnshipCallsign: cfg.Ownship.Callsign,
	}
}

func validateSettingsPayloadIn(p SettingsPayloadIn) error {
	if p.GDL90Dest == nil {
		return errors.New("gdl90_dest is required")
	}
	if strings.TrimSpace(*p.GDL90Dest) == "" {
		return errors.New("gdl90_dest must be non-empty")
	}
	if p.Interval == nil {
		return errors.New("interval is required")
	}
	if strings.TrimSpace(*p.Interval) == "" {
		return errors.New("interval must be non-empty")
	}
	if p.OwnshipICAO == nil {
		return errors.New("ownship_icao is required")
	}
	if strings.TrimSpace(*p.OwnshipICAO) == "" {
		return errors.New("ownship_icao must be non-empty")
	}
	if p.OwnshipCallsign == nil {
		return errors.New("ownship_callsign is required")
	}
	if strings.TrimSpace(*p.OwnshipCallsign) == "" {
		return errors.New("ownship_callsign must be non-empty")
	}
	return nil
}

func applySettingsPayload(cfg *config.Config, p SettingsPayloadIn) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if err := validateSettingsPayloadIn(p); err != nil {
		return err
	}

	cfg.GDL90.Dest = strings.TrimSpace(*p.GDL90Dest)

	intervalStr := strings.TrimSpace(*p.Interval)
	d, err := time.ParseDuration(intervalStr)
	if err != nil {
		return fmt.Errorf("invalid interval %q: %w", intervalStr, err)
	}
	cfg.GDL90.Interval = d

	cfg.Ownship.ICAO = strings.ToUpper(strings.TrimSpace(*p.OwnshipICAO))
	cfg.Ownship.Callsign = strings.TrimSpace(*p.OwnshipCallsign)
	return nil
}

type SettingsStore struct {
	ConfigPath string
	// Apply, when set, is called after validation and before saving.
	// If Apply returns an error, the config is not saved.
	// Apply is expected to make the new config effective immediately.
	Apply func(cfg config.Config) error
}

func (s SettingsStore) load() (config.Config, error) {
	return config.Load(s.ConfigPath)
}

func (s SettingsStore) save(cfg config.Config) error {
	if err := config.DefaultAndValidate(&cfg); err != nil {
		return err
	}
	b, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	// Write atomically to avoid corrupting config on crash/power loss.
	// Use a temp file in the same directory so os.Rename is atomic.
	dir := filepath.Dir(s.ConfigPath)
	tmp, err := os.CreateTemp(dir, filepath.Base(s.ConfigPath)+".tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.ConfigPath)
}

func (s SettingsStore) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(s.ConfigPath) == "" {
			http.Error(w, "settings not available (no config path)", http.StatusNotImplemented)
			return
		}

		switch r.Method {
		case http.MethodGet:
			cfg, err := s.load()
			if err != nil {
				http.Error(w, fmt.Sprintf("load failed: %v", err), http.StatusInternalServerError)
				return
			}
			payload := configToSettingsPayload(cfg)
			b, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				http.Error(w, "marshal failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(b)
			_, _ = w.Write([]byte("\n"))
			return

		case http.MethodPost:
			if ct := strings.TrimSpace(r.Header.Get("Content-Type")); ct != "application/json" {
				http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
				return
			}

			// Small config payload; cap to prevent unbounded reads.
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, fmt.Sprintf("read failed: %v", err), http.StatusBadRequest)
				return
			}
			p, err := decodeSettingsPayloadInStrict(body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			oldCfg, err := s.load()
			if err != nil {
				http.Error(w, fmt.Sprintf("load failed: %v", err), http.StatusInternalServerError)
				return
			}

			cfg := oldCfg
			if err := applySettingsPayload(&cfg, p); err != nil {
				http.Error(w, fmt.Sprintf("invalid settings: %v", err), http.StatusBadRequest)
				return
			}
			if err := config.DefaultAndValidate(&cfg); err != nil {
				http.Error(w, fmt.Sprintf("invalid config: %v", err), http.StatusBadRequest)
				return
			}

			if s.Apply != nil {
				if err := s.Apply(cfg); err != nil {
					http.Error(w, fmt.Sprintf("apply failed: %v", err), http.StatusBadRequest)
					return
				}
			}

			if err := s.save(cfg); err != nil {
				// Best-effort rollback to keep runtime consistent with disk.
				if s.Apply != nil {
					_ = s.Apply(oldCfg)
				}
				http.Error(w, fmt.Sprintf("save failed: %v", err), http.StatusInternalServerError)
				return
			}

			payload := configToSettingsPayload(cfg)
			b, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				http.Error(w, "marshal failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(b)
			_, _ = w.Write([]byte("\n"))
			return
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})

	return mux
}
