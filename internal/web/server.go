package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

//go:embed assets/*
var embeddedAssets embed.FS

// AHRSController optionally exposes calibration actions to the Web UI.
// Implementations should be safe to call concurrently.
type AHRSController interface {
	SetLevel() error
	ZeroDrift(ctx context.Context) error
	OrientForward(ctx context.Context) error
	OrientDone(ctx context.Context) error
	Orientation() (forwardAxis int, gravity [3]float64, gravityOK bool)
}

func Handler(status *Status, settings SettingsStore, logs *LogBuffer, ahrsCtl AHRSController) http.Handler {
	mux := http.NewServeMux()

	assetsFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		// Should never happen; keep server functional with API only.
		assetsFS = nil
	}

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		snap := status.Snapshot(time.Now().UTC())
		b, err := json.MarshalIndent(snap, "", "  ")
		if err != nil {
			http.Error(w, "marshal failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
		_, _ = w.Write([]byte("\n"))
	})

	// AHRS actions (optional).
	mux.HandleFunc("/api/ahrs/level", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ahrsCtl == nil {
			http.Error(w, "ahrs unavailable", http.StatusNotFound)
			return
		}
		if err := ahrsCtl.SetLevel(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"ok\":true}\n"))
	})

	mux.HandleFunc("/api/ahrs/zero-drift", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ahrsCtl == nil {
			http.Error(w, "ahrs unavailable", http.StatusNotFound)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := ahrsCtl.ZeroDrift(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"ok\":true}\n"))
	})

	mux.HandleFunc("/api/ahrs/orient/forward", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ahrsCtl == nil {
			http.Error(w, "ahrs unavailable", http.StatusNotFound)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := ahrsCtl.OrientForward(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"ok\":true}\n"))
	})

	mux.HandleFunc("/api/ahrs/orient/done", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ahrsCtl == nil {
			http.Error(w, "ahrs unavailable", http.StatusNotFound)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := ahrsCtl.OrientDone(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Best-effort: persist orientation to YAML config so it survives reboot.
		if strings.TrimSpace(settings.ConfigPath) != "" {
			forwardAxis, gravity, ok := ahrsCtl.Orientation()
			if ok && forwardAxis != 0 {
				cfg, err := settings.load()
				if err == nil {
					cfg.AHRS.Orientation.ForwardAxis = forwardAxis
					cfg.AHRS.Orientation.GravityInSensor = []float64{gravity[0], gravity[1], gravity[2]}
					_ = settings.save(cfg)
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"ok\":true}\n"))
	})

	// Scenario list API (used by the Settings UI).
	// Returns paths like "./configs/scenarios/edgecases.yaml".
	mux.HandleFunc("/api/scenarios", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		base := filepath.FromSlash("configs/scenarios")
		entries, err := os.ReadDir(base)
		if err != nil {
			// If the directory isn't present (e.g., minimal appliance build), return an empty list.
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("{\"paths\":[]}\n"))
			return
		}

		paths := make([]string, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			lower := strings.ToLower(name)
			if !(strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")) {
				continue
			}
			// Keep the returned value stable across platforms and consistent with config examples.
			paths = append(paths, "./configs/scenarios/"+name)
		}
		sort.Strings(paths)

		resp := struct {
			Paths []string `json:"paths"`
		}{Paths: paths}

		b, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			http.Error(w, "marshal failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
		_, _ = w.Write([]byte("\n"))
	})

	// Settings API (read/write YAML config). Changes are applied immediately when supported.
	// Kept intentionally small.
	mux.Handle("/api/settings", settings.Handler())

	if logs != nil {
		mux.Handle("/api/logs", logs.Handler())
	}

	// About.
	mux.Handle("/api/about", AboutHandler())

	if assetsFS != nil {
		fileServer := http.FileServer(http.FS(assetsFS))
		mux.Handle("/assets/", http.StripPrefix("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent stale UI assets during development.
			w.Header().Set("Cache-Control", "no-store")
			fileServer.ServeHTTP(w, r)
		})))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// SPA shell: serve the UI for / and any unknown paths (except /api/* and /assets/*).
		if r.URL.Path != "/" {
			if path.Dir(r.URL.Path) == "/api" || path.Dir(r.URL.Path) == "/assets" {
				http.NotFound(w, r)
				return
			}
		}

		if assetsFS == nil {
			// Fallback minimal page if embedding failed.
			snap := status.Snapshot(time.Now().UTC())
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, "<!doctype html><html><head><meta charset=\"utf-8\"><title>Stratux-NG</title></head><body>")
			_, _ = fmt.Fprintf(w, "<h1>Stratux-NG</h1>")
			_, _ = fmt.Fprintf(w, "<p>Web UI is unavailable. Use <a href=\"/api/status\">/api/status</a>.</p>")
			_, _ = fmt.Fprintf(w, "<pre>gdl90_dest=%s\ninterval=%s\nframes_sent_total=%d\nlast_tick_utc=%s</pre>",
				snap.GDL90Dest, snap.Interval, snap.FramesSentTotal, snap.LastTickUTC,
			)
			_, _ = fmt.Fprintf(w, "</body></html>")
			return
		}

		b, err := fs.ReadFile(assetsFS, "index.html")
		if err != nil {
			http.Error(w, "ui unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})

	return mux
}

func Serve(ctx context.Context, listenAddr string, status *Status, settings SettingsStore, logs *LogBuffer, ahrsCtl AHRSController) error {
	if status == nil {
		status = NewStatus()
	}

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           Handler(status, settings, logs, ahrsCtl),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MiB
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
