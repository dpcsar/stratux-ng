//go:build linux

package i2c

import (
	"os"
	"strings"
	"testing"
)

func TestDevTx_InvalidAddr(t *testing.T) {
	f, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("OpenFile /dev/null: %v", err)
	}
	defer f.Close()

	b := &Bus{f: f, path: "/dev/null"}

	{
		d := &Dev{bus: b, addr: 0}
		err := d.Write([]byte{0x00})
		if err == nil || !strings.Contains(err.Error(), "invalid i2c addr") {
			t.Fatalf("err=%v want invalid i2c addr", err)
		}
	}

	{
		d := &Dev{bus: b, addr: 0x80}
		err := d.Write([]byte{0x00})
		if err == nil || !strings.Contains(err.Error(), "invalid i2c addr") {
			t.Fatalf("err=%v want invalid i2c addr", err)
		}
	}
}

func TestDevTx_EmptyIsNoop(t *testing.T) {
	f, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("OpenFile /dev/null: %v", err)
	}
	defer f.Close()

	b := &Bus{f: f, path: "/dev/null"}
	d := &Dev{bus: b, addr: 0x68}

	n, err := d.tx(nil, nil)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if n != 0 {
		t.Fatalf("n=%d want 0", n)
	}
}
