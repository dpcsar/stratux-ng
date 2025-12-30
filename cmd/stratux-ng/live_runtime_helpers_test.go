package main

import (
	"reflect"
	"testing"
)

func TestIsDump1090Command(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		want bool
	}{
		{name: "empty", cmd: "", want: false},
		{name: "plain", cmd: "dump1090-fa", want: true},
		{name: "with_path", cmd: "/usr/local/bin/dump1090-fa", want: true},
		{name: "whitespace", cmd: "  /usr/bin/dump1090  ", want: true},
		{name: "case_insensitive", cmd: "DUMP1090-FA", want: true},
		{name: "not_prefix", cmd: "mydump1090-fa", want: false},
		{name: "other_binary", cmd: "dump978-fa", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDump1090Command(tc.cmd); got != tc.want {
				t.Fatalf("isDump1090Command(%q)=%v want %v", tc.cmd, got, tc.want)
			}
		})
	}
}

func TestIsDump978Command(t *testing.T) {
	cases := []struct {
		name string
		cmd  string
		want bool
	}{
		{name: "empty", cmd: "", want: false},
		{name: "plain", cmd: "dump978-fa", want: true},
		{name: "with_path", cmd: "/usr/local/bin/dump978-fa", want: true},
		{name: "whitespace", cmd: "  /usr/bin/dump978  ", want: true},
		{name: "case_insensitive", cmd: "DUMP978-FA", want: true},
		{name: "not_prefix", cmd: "mydump978-fa", want: false},
		{name: "other_binary", cmd: "dump1090-fa", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isDump978Command(tc.cmd); got != tc.want {
				t.Fatalf("isDump978Command(%q)=%v want %v", tc.cmd, got, tc.want)
			}
		})
	}
}

func TestUpsertDump1090DeviceArgs(t *testing.T) {
	t.Run("no_change_when_net_only", func(t *testing.T) {
		in := []string{"--net-only", "--device", "SHOULD_NOT_CHANGE"}
		out := upsertDump1090DeviceArgs(append([]string(nil), in...), "stx:1090:0", nil)
		if !reflect.DeepEqual(out, in) {
			t.Fatalf("args changed unexpectedly: got=%q want=%q", out, in)
		}
	})

	t.Run("no_change_when_ifile", func(t *testing.T) {
		in := []string{"--ifile", "foo.bin", "--device", "SHOULD_NOT_CHANGE"}
		out := upsertDump1090DeviceArgs(append([]string(nil), in...), "stx:1090:0", nil)
		if !reflect.DeepEqual(out, in) {
			t.Fatalf("args changed unexpectedly: got=%q want=%q", out, in)
		}
	})

	t.Run("inserts_device_when_missing", func(t *testing.T) {
		in := []string{"--device-type", "rtlsdr"}
		out := upsertDump1090DeviceArgs(append([]string(nil), in...), "stx:1090:0", nil)
		want := []string{"--device-type", "rtlsdr", "--device", "stx:1090:0"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got=%q want=%q", out, want)
		}
	})

	t.Run("overwrites_existing_device_value", func(t *testing.T) {
		in := []string{"--device-type", "rtlsdr", "--device", "OLD"}
		out := upsertDump1090DeviceArgs(append([]string(nil), in...), "stx:1090:0", nil)
		want := []string{"--device-type", "rtlsdr", "--device", "stx:1090:0"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got=%q want=%q", out, want)
		}
	})

	t.Run("overwrites_existing_device_equals_form", func(t *testing.T) {
		in := []string{"--device-type", "rtlsdr", "--device=OLD"}
		out := upsertDump1090DeviceArgs(append([]string(nil), in...), "stx:1090:0", nil)
		want := []string{"--device-type", "rtlsdr", "--device=stx:1090:0"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got=%q want=%q", out, want)
		}
	})

	t.Run("uses_index_when_no_serial", func(t *testing.T) {
		idx := 7
		in := []string{"--device-type", "rtlsdr"}
		out := upsertDump1090DeviceArgs(append([]string(nil), in...), " ", &idx)
		want := []string{"--device-type", "rtlsdr", "--device", "7"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got=%q want=%q", out, want)
		}
	})

	t.Run("no_change_when_no_serial_and_no_index", func(t *testing.T) {
		in := []string{"--device-type", "rtlsdr"}
		out := upsertDump1090DeviceArgs(append([]string(nil), in...), "", nil)
		if !reflect.DeepEqual(out, in) {
			t.Fatalf("got=%q want=%q", out, in)
		}
	})
}

func TestUpsertDump978DeviceArgs(t *testing.T) {
	t.Run("no_change_when_stratuxv3", func(t *testing.T) {
		in := []string{"--stratuxv3", "/dev/ttyUSB0", "--sdr", "SHOULD_NOT_CHANGE"}
		out := upsertDump978DeviceArgs(append([]string(nil), in...), "stx:978:0")
		if !reflect.DeepEqual(out, in) {
			t.Fatalf("args changed unexpectedly: got=%q want=%q", out, in)
		}
	})

	t.Run("no_change_when_stdin", func(t *testing.T) {
		in := []string{"--stdin", "--sdr", "SHOULD_NOT_CHANGE"}
		out := upsertDump978DeviceArgs(append([]string(nil), in...), "stx:978:0")
		if !reflect.DeepEqual(out, in) {
			t.Fatalf("args changed unexpectedly: got=%q want=%q", out, in)
		}
	})

	t.Run("no_change_when_file", func(t *testing.T) {
		in := []string{"--file", "in.raw", "--sdr", "SHOULD_NOT_CHANGE"}
		out := upsertDump978DeviceArgs(append([]string(nil), in...), "stx:978:0")
		if !reflect.DeepEqual(out, in) {
			t.Fatalf("args changed unexpectedly: got=%q want=%q", out, in)
		}
	})

	t.Run("inserts_sdr_when_missing", func(t *testing.T) {
		in := []string{"--json-port", "30978"}
		out := upsertDump978DeviceArgs(append([]string(nil), in...), "stx:978:0")
		want := []string{"--json-port", "30978", "--sdr", "driver=rtlsdr,serial=stx:978:0"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got=%q want=%q", out, want)
		}
	})

	t.Run("overwrites_existing_sdr_value", func(t *testing.T) {
		in := []string{"--sdr", "driver=rtlsdr,serial=OLD", "--json-port", "30978"}
		out := upsertDump978DeviceArgs(append([]string(nil), in...), "stx:978:0")
		want := []string{"--sdr", "driver=rtlsdr,serial=stx:978:0", "--json-port", "30978"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got=%q want=%q", out, want)
		}
	})

	t.Run("overwrites_existing_sdr_equals_form", func(t *testing.T) {
		in := []string{"--sdr=driver=rtlsdr,serial=OLD", "--json-port", "30978"}
		out := upsertDump978DeviceArgs(append([]string(nil), in...), "stx:978:0")
		want := []string{"--sdr=driver=rtlsdr,serial=stx:978:0", "--json-port", "30978"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got=%q want=%q", out, want)
		}
	})

	t.Run("auto_serial_becomes_driver_only", func(t *testing.T) {
		in := []string{"--json-port", "30978"}
		out := upsertDump978DeviceArgs(append([]string(nil), in...), "auto")
		want := []string{"--json-port", "30978", "--sdr", "driver=rtlsdr"}
		if !reflect.DeepEqual(out, want) {
			t.Fatalf("got=%q want=%q", out, want)
		}
	})
}
