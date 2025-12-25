//go:build !linux

package web

import (
	"context"
	"errors"
)

func scanWiFiNetworks(_ context.Context, _ string) ([]WiFiScanNetwork, error) {
	return nil, errors.New("wifi scan unsupported on this platform")
}
