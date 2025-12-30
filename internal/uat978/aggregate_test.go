package uat978

import (
	"testing"
	"time"
)

func TestAggregator_TowerCountsAndSignalAveraging(t *testing.T) {
	agg := NewAggregator(AggregatorConfig{MaxTowers: 10, MaxText: 10, MaxRows: 50})
	now := time.Unix(1_000_000, 0).UTC()

	// Same tower, three messages in the same second; only two have signal.
	d := DecodedUplink{TowerLatDeg: 37.0, TowerLonDeg: -122.0}
	agg.Add(now, d, -10.0, true)
	agg.Add(now, d, -20.0, true)
	agg.Add(now, d, 0.0, false)

	towers, _ := agg.Snapshot(now)
	if len(towers) != 1 {
		t.Fatalf("expected 1 tower, got %d", len(towers))
	}
	got := towers[0]
	if got.MessagesLastMin != 3 {
		t.Fatalf("expected msg/min=3, got %d", got.MessagesLastMin)
	}
	if !got.HasSignalStrength {
		t.Fatalf("expected HasSignalStrength")
	}

	// Average should be over the two signaled messages only.
	wantAvg := (-10.0 + -20.0) / 2.0
	if got.SignalAvg1MinDb != wantAvg {
		t.Fatalf("expected avg=%.3f, got %.3f", wantAvg, got.SignalAvg1MinDb)
	}
	if got.SignalMaxDb != -10.0 {
		t.Fatalf("expected max=-10.0, got %.3f", got.SignalMaxDb)
	}
}

func TestAggregator_WeatherProductsAndTextRing(t *testing.T) {
	agg := NewAggregator(AggregatorConfig{MaxTowers: 10, MaxText: 3, MaxRows: 50})
	base := time.Unix(1_000_000, 0).UTC()

	// Add product hits and text lines.
	d1 := DecodedUplink{TowerLatDeg: 1, TowerLonDeg: 2, ProductIDs: []uint32{413, 63}, TextReports: []string{"A", "B"}}
	agg.Add(base, d1, -5, true)

	d2 := DecodedUplink{TowerLatDeg: 1, TowerLonDeg: 2, ProductIDs: []uint32{413}, TextReports: []string{"C", "D"}}
	agg.Add(base.Add(1*time.Second), d2, -6, true)

	_, wx := agg.Snapshot(base.Add(1 * time.Second))
	if len(wx.Products) == 0 {
		t.Fatalf("expected products")
	}

	// Text ring is newest-first and capped at MaxText=3.
	if len(wx.Text) != 3 {
		t.Fatalf("expected 3 text entries, got %d", len(wx.Text))
	}
	if wx.Text[0].Text != "D" {
		t.Fatalf("expected newest text 'D', got %q", wx.Text[0].Text)
	}
	if wx.Text[1].Text != "C" {
		t.Fatalf("expected next text 'C', got %q", wx.Text[1].Text)
	}
	if wx.Text[2].Text != "B" {
		t.Fatalf("expected oldest kept text 'B', got %q", wx.Text[2].Text)
	}
}
