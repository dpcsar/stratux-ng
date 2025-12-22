package bmp280

import (
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

type fakeI2C struct {
	// Simple register model.
	regs map[byte][]byte

	// Calibration read behavior.
	calibReads int
	calibSeq   [][]byte

	writes []writeOp
}

type writeOp struct {
	reg byte
	val byte
}

func (f *fakeI2C) ReadRegU8(reg byte) (byte, error) {
	b, ok := f.regs[reg]
	if !ok || len(b) < 1 {
		return 0, errors.New("no reg")
	}
	return b[0], nil
}

func (f *fakeI2C) ReadReg(reg byte, dst []byte) error {
	if reg == regCalib00 {
		f.calibReads++
		idx := f.calibReads - 1
		if idx < len(f.calibSeq) {
			copy(dst, f.calibSeq[idx])
			return nil
		}
		// Default to zeros.
		for i := range dst {
			dst[i] = 0
		}
		return nil
	}

	b, ok := f.regs[reg]
	if !ok {
		return errors.New("no reg")
	}
	copy(dst, b)
	return nil
}

func (f *fakeI2C) WriteReg(reg, value byte) error {
	f.writes = append(f.writes, writeOp{reg: reg, val: value})
	return nil
}

func TestNew_RetriesCalibrationAfterReset(t *testing.T) {
	oldSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = oldSleep })

	calibZero := make([]byte, calibLen)
	calibOK := make([]byte, calibLen)
	binary.LittleEndian.PutUint16(calibOK[0:2], 27504) // digT1
	binary.LittleEndian.PutUint16(calibOK[6:8], 36477) // digP1
	binary.LittleEndian.PutUint16(calibOK[2:4], 26435) // digT2 (non-zero, optional)
	binary.LittleEndian.PutUint16(calibOK[8:10], 2855) // digP2 (non-zero, optional)

	f := &fakeI2C{
		regs: map[byte][]byte{
			regID: {chipIDBMP280},
		},
		calibSeq: [][]byte{calibZero, calibOK},
	}

	_, err := newWithIO(f)
	if err != nil {
		t.Fatalf("expected New to succeed, got %v", err)
	}
	if f.calibReads < 2 {
		t.Fatalf("expected calibration to be retried, reads=%d", f.calibReads)
	}
}

func TestNew_FailsOnInvalidCalibration(t *testing.T) {
	oldSleep := sleep
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = oldSleep })

	calibZero := make([]byte, calibLen)
	f := &fakeI2C{
		regs: map[byte][]byte{
			regID: {chipIDBMP280},
		},
		calibSeq: [][]byte{calibZero, calibZero, calibZero},
	}

	_, err := newWithIO(f)
	if err == nil {
		t.Fatalf("expected invalid calibration error")
	}
}
