package web

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
)

type WiFiScanNetwork struct {
	SSID     string `json:"ssid"`
	Security string `json:"security,omitempty"`
	Signal   int    `json:"signal,omitempty"`
}

type wifiScanResponse struct {
	Networks  []WiFiScanNetwork `json:"networks"`
	LastError string            `json:"last_error,omitempty"`
}

type wifiScanFunc func(ctx context.Context, iface string) ([]WiFiScanNetwork, error)

// wifiScanFn can be overridden in tests.
var wifiScanFn wifiScanFunc = scanWiFiNetworks

func WiFiScanHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		iface := strings.TrimSpace(r.URL.Query().Get("iface"))

		nets, err := wifiScanFn(r.Context(), iface)
		resp := wifiScanResponse{Networks: nets}
		if err != nil {
			resp.LastError = err.Error()
		}

		// Stable ordering for the UI.
		sort.Slice(resp.Networks, func(i, j int) bool {
			a := resp.Networks[i]
			b := resp.Networks[j]
			if a.Signal != b.Signal {
				return a.Signal > b.Signal
			}
			return a.SSID < b.SSID
		})

		b, err2 := json.MarshalIndent(resp, "", "  ")
		if err2 != nil {
			http.Error(w, "marshal failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
		_, _ = w.Write([]byte("\n"))
	})
}
