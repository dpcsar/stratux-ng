package sdr

import "testing"

func TestParseRTLTestOutput(t *testing.T) {
	out := `Found 2 device(s):
  0:  Realtek, RTL2838UHIDIR, SN: DK0GIZ2T
  1:  Realtek, RTL2838UHIDIR, SN: stratux:1090

Using device 0: Generic RTL2832U OEM
`
	devs := ParseRTLTestOutput(out)
	if len(devs) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devs))
	}
	if devs[0].Index != 0 || devs[0].Serial != "DK0GIZ2T" {
		t.Fatalf("device0 mismatch: %+v", devs[0])
	}
	if devs[1].Index != 1 || devs[1].Serial != "stratux:1090" {
		t.Fatalf("device1 mismatch: %+v", devs[1])
	}
}

func TestAutoAssign1090And978(t *testing.T) {
	devs := []RTLSDRDevice{{Index: 0, Serial: "DK0GIZ2T"}, {Index: 1, Serial: "stratux:1090"}}
	adsb, uat := AutoAssign1090And978(devs)
	if adsb == nil || adsb.Serial != "stratux:1090" {
		t.Fatalf("expected 1090=stratux:1090, got %+v", adsb)
	}
	if uat == nil || uat.Serial != "DK0GIZ2T" {
		t.Fatalf("expected 978=DK0GIZ2T, got %+v", uat)
	}
}

func TestUpsertFlagValue(t *testing.T) {
	args := []string{"--foo", "1", "--bar=2"}
	args = UpsertFlagValue(args, "--foo", "9")
	args = UpsertFlagValue(args, "--bar", "7")
	args = UpsertFlagValue(args, "--baz", "3")

	want := []string{"--foo", "9", "--bar=7", "--baz", "3"}
	if len(args) != len(want) {
		t.Fatalf("len mismatch: %v vs %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("arg[%d]=%q want %q", i, args[i], want[i])
		}
	}
}

func TestBuildDump978SoapyRTLSDRArg(t *testing.T) {
	if got := BuildDump978SoapyRTLSDRArg(""); got != "driver=rtlsdr" {
		t.Fatalf("empty serial got %q", got)
	}
	if got := BuildDump978SoapyRTLSDRArg("auto"); got != "driver=rtlsdr" {
		t.Fatalf("auto serial got %q", got)
	}
	if got := BuildDump978SoapyRTLSDRArg("stx:978:0"); got != "driver=rtlsdr,serial=stx:978:0" {
		t.Fatalf("serial got %q", got)
	}
}
