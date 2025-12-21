package main

import (
	"testing"
	"time"

	"stratux-ng/internal/config"
	"stratux-ng/internal/gdl90"
	"stratux-ng/internal/sim"
)

func TestBuildGDL90FramesFromScenario_MessageSet(t *testing.T) {
	scriptYAML := []byte(`
version: 1
duration: 2s
ownship:
  icao: "F000AA"
  callsign: "OWN123"
  gps_horizontal_accuracy_m: 10
  keyframes:
    - t: 0s
      lat_deg: 45
      lon_deg: -90
      alt_feet: 3000
      ground_kt: 0
      track_deg: 350
    - t: 2s
      lat_deg: 45
      lon_deg: -90
      alt_feet: 3000
      ground_kt: 0
      track_deg: 10
traffic:
  - icao: "F10001"
    callsign: "TGT0001"
    keyframes:
      - t: 0s
        lat_deg: 45
        lon_deg: -90
        alt_feet: 3200
        ground_kt: 0
        track_deg: 90
  - icao: "F10002"
    callsign: "TGT0002"
    keyframes:
      - t: 0s
        lat_deg: 45
        lon_deg: -90
        alt_feet: 3400
        ground_kt: 120
        track_deg: 90
`)

	script, err := sim.ParseScenarioScriptYAML(scriptYAML)
	if err != nil {
		t.Fatalf("ParseScenarioScriptYAML: %v", err)
	}
	scn, err := sim.NewScenario(script)
	if err != nil {
		t.Fatalf("NewScenario: %v", err)
	}

	cfg := config.Config{Sim: config.SimConfig{Scenario: config.ScenarioSimConfig{Enable: true, Loop: true}}}

	ownICAO, err := gdl90.ParseICAOHex(script.Ownship.ICAO)
	if err != nil {
		t.Fatalf("ParseICAOHex ownship: %v", err)
	}
	trafficICAO := make([][3]byte, len(script.Traffic))
	for i := range script.Traffic {
		p, err := gdl90.ParseICAOHex(script.Traffic[i].ICAO)
		if err != nil {
			t.Fatalf("ParseICAOHex traffic[%d]: %v", i, err)
		}
		trafficICAO[i] = p
	}

	now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	frames := buildGDL90FramesFromScenario(cfg, now, 0, scn, ownICAO, trafficICAO)
	if len(frames) == 0 {
		t.Fatalf("expected frames")
	}

	counts := map[byte]int{}
	ffSub := map[byte]int{}
	for _, f := range frames {
		msg := unframeForMsg(t, f)
		counts[msg[0]]++
		if msg[0] == 0x65 && len(msg) >= 2 {
			ffSub[msg[1]]++
		}
	}

	if counts[0x00] != 1 {
		t.Fatalf("expected 1 heartbeat (0x00), got %d", counts[0x00])
	}
	if counts[0xCC] != 1 {
		t.Fatalf("expected 1 stratux heartbeat (0xCC), got %d", counts[0xCC])
	}
	if ffSub[0x00] != 1 {
		t.Fatalf("expected 1 device ID (0x65/0x00), got %d", ffSub[0x00])
	}
	if ffSub[0x01] != 1 {
		t.Fatalf("expected 1 AHRS message (0x65/0x01), got %d", ffSub[0x01])
	}
	if counts[0x0A] != 1 {
		t.Fatalf("expected 1 ownship report (0x0A), got %d", counts[0x0A])
	}
	if counts[0x0B] != 1 {
		t.Fatalf("expected 1 ownship geometric alt (0x0B), got %d", counts[0x0B])
	}
	if counts[0x14] != 2 {
		t.Fatalf("expected 2 traffic reports (0x14), got %d", counts[0x14])
	}
}
