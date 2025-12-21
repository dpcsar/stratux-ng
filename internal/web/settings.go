package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"stratux-ng/internal/config"
)

type SettingsPayload struct {
	GDL90Dest string `json:"gdl90_dest"`
	Interval  string `json:"interval"`

	ScenarioEnable       bool   `json:"scenario_enable"`
	ScenarioPath         string `json:"scenario_path"`
	ScenarioStartTimeUTC string `json:"scenario_start_time_utc"`
	ScenarioLoop         bool   `json:"scenario_loop"`

	OwnshipEnable bool `json:"ownship_enable"`
	TrafficEnable bool `json:"traffic_enable"`

	WebEnable bool   `json:"web_enable"`
	WebListen string `json:"web_listen"`
}

func configToSettingsPayload(cfg config.Config) SettingsPayload {
	return SettingsPayload{
		GDL90Dest: cfg.GDL90.Dest,
		Interval:  cfg.GDL90.Interval.String(),

		ScenarioEnable:       cfg.Sim.Scenario.Enable,
		ScenarioPath:         cfg.Sim.Scenario.Path,
		ScenarioStartTimeUTC: cfg.Sim.Scenario.StartTimeUTC,
		ScenarioLoop:         cfg.Sim.Scenario.Loop,

		OwnshipEnable: cfg.Sim.Ownship.Enable,
		TrafficEnable: cfg.Sim.Traffic.Enable,

		WebEnable: cfg.Web.Enable,
		WebListen: cfg.Web.Listen,
	}
}

func applySettingsPayload(cfg *config.Config, p SettingsPayload) {
	if cfg == nil {
		return
	}
	cfg.GDL90.Dest = strings.TrimSpace(p.GDL90Dest)
	// Interval is intentionally not applied here; changing ticker intervals live
	// is out of scope for the first web UI iteration.

	cfg.Sim.Scenario.Enable = p.ScenarioEnable
	cfg.Sim.Scenario.Path = strings.TrimSpace(p.ScenarioPath)
	cfg.Sim.Scenario.StartTimeUTC = strings.TrimSpace(p.ScenarioStartTimeUTC)
	cfg.Sim.Scenario.Loop = p.ScenarioLoop

	cfg.Sim.Ownship.Enable = p.OwnshipEnable
	cfg.Sim.Traffic.Enable = p.TrafficEnable

	cfg.Web.Enable = p.WebEnable
	cfg.Web.Listen = strings.TrimSpace(p.WebListen)
}

type SettingsStore struct {
	ConfigPath string
}

func (s SettingsStore) load() (config.Config, error) {
	b, err := os.ReadFile(s.ConfigPath)
	if err != nil {
		return config.Config{}, err
	}
	var cfg config.Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return config.Config{}, err
	}
	if err := config.DefaultAndValidate(&cfg); err != nil {
		return config.Config{}, err
	}
	return cfg, nil
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
			// Small config payload; cap to prevent unbounded reads.
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
			var p SettingsPayload
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			if err := dec.Decode(&p); err != nil {
				http.Error(w, fmt.Sprintf("invalid json: %v", err), http.StatusBadRequest)
				return
			}

			cfg, err := s.load()
			if err != nil {
				http.Error(w, fmt.Sprintf("load failed: %v", err), http.StatusInternalServerError)
				return
			}
			applySettingsPayload(&cfg, p)
			if err := s.save(cfg); err != nil {
				http.Error(w, fmt.Sprintf("save failed: %v", err), http.StatusBadRequest)
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
