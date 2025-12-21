package web

import (
	"encoding/json"
	"net/http"
	"runtime"
	"runtime/debug"
	"time"
)

type AboutResponse struct {
	Service    string `json:"service"`
	NowUTC     string `json:"now_utc"`
	GoVersion  string `json:"go_version"`
	ModulePath string `json:"module_path,omitempty"`
	Version    string `json:"version,omitempty"`
	Commit     string `json:"commit,omitempty"`
	Dirty      bool   `json:"dirty,omitempty"`
	BuildTime  string `json:"build_time,omitempty"`
}

func AboutHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		resp := AboutResponse{
			Service:   "stratux-ng",
			NowUTC:    time.Now().UTC().Format(time.RFC3339Nano),
			GoVersion: runtime.Version(),
		}

		if bi, ok := debug.ReadBuildInfo(); ok && bi != nil {
			resp.ModulePath = bi.Main.Path
			resp.Version = bi.Main.Version
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					resp.Commit = s.Value
				case "vcs.modified":
					resp.Dirty = s.Value == "true"
				case "vcs.time":
					resp.BuildTime = s.Value
				}
			}
		}

		b, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			http.Error(w, "marshal failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
		_, _ = w.Write([]byte("\n"))
	})
}
