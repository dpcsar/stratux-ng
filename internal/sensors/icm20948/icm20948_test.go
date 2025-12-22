package icm20948

import (
	"errors"
	"testing"
	"time"
)

type fakeI2C struct {
	regs   map[byte][]byte
	writes []writeOp

	// Optional overrides.
	readErrFor map[byte]error
}

type writeOp struct {
	reg byte
	val byte
}

func (f *fakeI2C) ReadRegU8(reg byte) (byte, error) {
	if err := f.readErrFor[reg]; err != nil {
		return 0, err
	}
	b := f.regs[reg]
	if len(b) < 1 {
		return 0, errors.New("no reg")
	}
	return b[0], nil
}

func (f *fakeI2C) ReadReg(reg byte, dst []byte) error {
	if err := f.readErrFor[reg]; err != nil {
		return err
	}
	b := f.regs[reg]
	if len(b) < len(dst) {
		return errors.New("short reg")
	}
	copy(dst, b[:len(dst)])
	return nil
}

func (f *fakeI2C) WriteReg(reg, value byte) error {
	f.writes = append(f.writes, writeOp{reg: reg, val: value})
	return nil
}

func TestNew_WhoAmIMismatch(t *testing.T) {
	oldSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = oldSleep })

	f := &fakeI2C{regs: map[byte][]byte{regWhoAmI: {0x00}}}
	_, err := newWithIO(f)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNew_WritesExpectedInitRegisters(t *testing.T) {
	oldSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = oldSleep })

	f := &fakeI2C{regs: map[byte][]byte{regWhoAmI: {whoAmIVal}}}
	_, err := newWithIO(f)
	if err != nil {
		t.Fatalf("newWithIO: %v", err)
	}

	// Ensure we wrote reset + wake.
	var sawReset, sawWake bool
	for _, w := range f.writes {
		if w.reg == regPwrMgmt1 && w.val == bitReset {
			sawReset = true
		}
		if w.reg == regPwrMgmt1 && w.val == 0x01 {
			sawWake = true
		}
	}
	if !sawReset {
		t.Fatalf("expected reset write to PWR_MGMT_1")
	}
	if !sawWake {
		t.Fatalf("expected wake write to PWR_MGMT_1")
	}

	// Ensure we selected bank 2 at least once.
	var sawBank2 bool
	for _, w := range f.writes {
		if w.reg == regBankSel && w.val == (bank2<<4) {
			sawBank2 = true
			break
		}
	}
	if !sawBank2 {
		t.Fatalf("expected bank2 select write")
	}
}

func TestRead_ScalesAccelAndGyro(t *testing.T) {
	oldSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = oldSleep })

	// ax=16384 -> 2g when full-scale=4g (4/32768)
	// gx=16384 -> 125 dps when full-scale=250dps (250/32768)
	f := &fakeI2C{regs: map[byte][]byte{regWhoAmI: {whoAmIVal}}}

	// Register block starting at ACCEL_XOUT_H.
	f.regs[regAccelXoutH] = []byte{
		0x40, 0x00, // ax
		0x00, 0x00, // ay
		0xC0, 0x00, // az = -16384 -> -2g
		0x40, 0x00, // gx
		0x00, 0x00, // gy
		0xC0, 0x00, // gz = -16384 -> -125 dps
	}

	d, err := newWithIO(f)
	if err != nil {
		t.Fatalf("newWithIO: %v", err)
	}

	s, err := d.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if s.Ax < 1.99 || s.Ax > 2.01 {
		t.Fatalf("Ax=%v want ~2.0", s.Ax)
	}
	if s.Az > -1.99 || s.Az < -2.01 {
		t.Fatalf("Az=%v want ~-2.0", s.Az)
	}
	if s.Gx < 124.9 || s.Gx > 125.1 {
		t.Fatalf("Gx=%v want ~125", s.Gx)
	}
	if s.Gz > -124.9 || s.Gz < -125.1 {
		t.Fatalf("Gz=%v want ~-125", s.Gz)
	}
}
