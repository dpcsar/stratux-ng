package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"stratux-ng/internal/config"
)

// AircraftProfilePayload is one named aircraft profile as exposed over the API.
type AircraftProfilePayload struct {
	Name     string `json:"name"`
	ICAO     string `json:"icao"`
	Callsign string `json:"callsign"`
}

// AircraftSettingsPayload is the full aircraft-profile list plus which one is
// active. GET/POST both use this shape (POST is a strict full replace, same
// convention as /api/settings: no partial updates).
type AircraftSettingsPayload struct {
	Active   string                   `json:"active"`
	Profiles []AircraftProfilePayload `json:"profiles"`
}

var aircraftTopKeys = []string{"active", "profiles"}
var aircraftProfileKeys = []string{"name", "icao", "callsign"}

// strictJSONObjectFields parses a single JSON object, rejecting unknown keys,
// duplicate keys, and null values, and returns the raw value for each allowed
// key that was present.
func strictJSONObjectFields(raw []byte, allowed []string) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))

	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return nil, errors.New("invalid json: expected object")
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, k := range allowed {
		allowedSet[k] = struct{}{}
	}
	out := make(map[string]json.RawMessage, len(allowed))

	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("invalid json: %w", err)
		}
		key, ok := kt.(string)
		if !ok {
			return nil, errors.New("invalid json: expected string key")
		}
		if _, ok := allowedSet[key]; !ok {
			return nil, fmt.Errorf("invalid json: unknown key %q", key)
		}
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("invalid json: duplicate key %q", key)
		}

		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, fmt.Errorf("invalid json: %w", err)
		}
		if strings.TrimSpace(string(val)) == "null" {
			return nil, fmt.Errorf("invalid json: %q cannot be null", key)
		}
		out[key] = val
	}

	end, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}
	if delim, ok := end.(json.Delim); !ok || delim != '}' {
		return nil, errors.New("invalid json: expected end of object")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("invalid json: trailing data")
	}

	return out, nil
}

func decodeAircraftPayloadInStrict(body []byte) (AircraftSettingsPayload, error) {
	top, err := strictJSONObjectFields(body, aircraftTopKeys)
	if err != nil {
		return AircraftSettingsPayload{}, err
	}
	for _, k := range aircraftTopKeys {
		if _, ok := top[k]; !ok {
			return AircraftSettingsPayload{}, fmt.Errorf("invalid json: missing required key %q", k)
		}
	}

	var out AircraftSettingsPayload
	if err := json.Unmarshal(top["active"], &out.Active); err != nil {
		return AircraftSettingsPayload{}, fmt.Errorf("invalid json: active: %w", err)
	}

	var rawProfiles []json.RawMessage
	if err := json.Unmarshal(top["profiles"], &rawProfiles); err != nil {
		return AircraftSettingsPayload{}, fmt.Errorf("invalid json: profiles: %w", err)
	}

	out.Profiles = make([]AircraftProfilePayload, 0, len(rawProfiles))
	for i, rp := range rawProfiles {
		fields, err := strictJSONObjectFields(rp, aircraftProfileKeys)
		if err != nil {
			return AircraftSettingsPayload{}, fmt.Errorf("invalid json: profiles[%d]: %w", i, err)
		}
		for _, k := range aircraftProfileKeys {
			if _, ok := fields[k]; !ok {
				return AircraftSettingsPayload{}, fmt.Errorf("invalid json: profiles[%d] missing required key %q", i, k)
			}
		}
		var p AircraftProfilePayload
		if err := json.Unmarshal(fields["name"], &p.Name); err != nil {
			return AircraftSettingsPayload{}, fmt.Errorf("invalid json: profiles[%d].name: %w", i, err)
		}
		if err := json.Unmarshal(fields["icao"], &p.ICAO); err != nil {
			return AircraftSettingsPayload{}, fmt.Errorf("invalid json: profiles[%d].icao: %w", i, err)
		}
		if err := json.Unmarshal(fields["callsign"], &p.Callsign); err != nil {
			return AircraftSettingsPayload{}, fmt.Errorf("invalid json: profiles[%d].callsign: %w", i, err)
		}
		out.Profiles = append(out.Profiles, p)
	}

	return out, nil
}

func configToAircraftPayload(cfg config.Config) AircraftSettingsPayload {
	out := AircraftSettingsPayload{
		Active:   cfg.Aircraft.Active,
		Profiles: make([]AircraftProfilePayload, 0, len(cfg.Aircraft.Profiles)),
	}
	for _, p := range cfg.Aircraft.Profiles {
		out.Profiles = append(out.Profiles, AircraftProfilePayload{
			Name:     p.Name,
			ICAO:     p.ICAO,
			Callsign: p.Callsign,
		})
	}
	return out
}

func validateAircraftPayload(p AircraftSettingsPayload) error {
	if len(p.Profiles) == 0 {
		return errors.New("profiles must contain at least one aircraft profile")
	}
	seen := make(map[string]struct{}, len(p.Profiles))
	for i, prof := range p.Profiles {
		name := strings.TrimSpace(prof.Name)
		if name == "" {
			return fmt.Errorf("profiles[%d].name must be non-empty", i)
		}
		if strings.TrimSpace(prof.ICAO) == "" {
			return fmt.Errorf("profiles[%d].icao must be non-empty", i)
		}
		if strings.TrimSpace(prof.Callsign) == "" {
			return fmt.Errorf("profiles[%d].callsign must be non-empty", i)
		}
		key := strings.ToLower(name)
		if _, dup := seen[key]; dup {
			return fmt.Errorf("duplicate profile name %q", name)
		}
		seen[key] = struct{}{}
	}
	active := strings.TrimSpace(p.Active)
	if active == "" {
		return errors.New("active must be non-empty")
	}
	if _, ok := seen[strings.ToLower(active)]; !ok {
		return fmt.Errorf("active %q does not match any profile name", active)
	}
	return nil
}

func applyAircraftPayload(cfg *config.Config, p AircraftSettingsPayload) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if err := validateAircraftPayload(p); err != nil {
		return err
	}

	profiles := make([]config.AircraftProfile, 0, len(p.Profiles))
	for _, prof := range p.Profiles {
		profiles = append(profiles, config.AircraftProfile{
			Name:     strings.TrimSpace(prof.Name),
			ICAO:     strings.ToUpper(strings.TrimSpace(prof.ICAO)),
			Callsign: strings.TrimSpace(prof.Callsign),
		})
	}
	cfg.Aircraft.Profiles = profiles
	cfg.Aircraft.Active = strings.TrimSpace(p.Active)
	return nil
}

func (s SettingsStore) handleAircraft(w http.ResponseWriter, r *http.Request) {
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
		payload := configToAircraftPayload(cfg)
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

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("read failed: %v", err), http.StatusBadRequest)
			return
		}
		p, err := decodeAircraftPayloadInStrict(body)
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
		if err := applyAircraftPayload(&cfg, p); err != nil {
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
			if s.Apply != nil {
				_ = s.Apply(oldCfg)
			}
			http.Error(w, fmt.Sprintf("save failed: %v", err), http.StatusInternalServerError)
			return
		}

		payload := configToAircraftPayload(cfg)
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
}
