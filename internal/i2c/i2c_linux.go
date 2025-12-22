//go:build linux

package i2c

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Minimal Linux I2C implementation backed by /dev/i2c-*.
//
// We use I2C_RDWR so we can do a combined write+read (repeated start), which
// many sensors require for register reads.

const (
	i2cMrd  = 0x0001
	i2cRdwr = 0x0707
)

type msg struct {
	addr  uint16
	flags uint16
	len   uint16
	buf   uintptr
}

type rdwrData struct {
	msgs  uintptr
	nmsgs uint32
}

// Bus is an opened I2C bus (e.g., /dev/i2c-1).
//
// It is safe to create multiple Dev handles from a single Bus.
// Bus itself is not safe for concurrent transfers; coordinate at a higher
// level if you need concurrency.
//
//nolint:revive // simple device abstraction.
type Bus struct {
	f    *os.File
	path string
}

func Open(path string) (*Bus, error) {
	path = filepath.Clean(path)
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return &Bus{f: f, path: path}, nil
}

func (b *Bus) Close() error {
	if b == nil || b.f == nil {
		return nil
	}
	err := b.f.Close()
	b.f = nil
	return err
}

func (b *Bus) Dev(addr uint16) *Dev {
	if b == nil {
		return nil
	}
	return &Dev{bus: b, addr: addr}
}

// Dev represents a device at a 7-bit I2C address.
//
//nolint:revive // minimal.
type Dev struct {
	bus  *Bus
	addr uint16
}

func (d *Dev) Write(p []byte) error {
	_, err := d.tx(p, nil)
	return err
}

func (d *Dev) Read(p []byte) error {
	_, err := d.tx(nil, p)
	return err
}

func (d *Dev) WriteRead(w, r []byte) error {
	_, err := d.tx(w, r)
	return err
}

func (d *Dev) ReadReg(reg byte, dst []byte) error {
	return d.WriteRead([]byte{reg}, dst)
}

func (d *Dev) ReadRegU8(reg byte) (byte, error) {
	var b [1]byte
	if err := d.ReadReg(reg, b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

func (d *Dev) WriteReg(reg, value byte) error {
	return d.Write([]byte{reg, value})
}

func (d *Dev) tx(w, r []byte) (int, error) {
	if d == nil || d.bus == nil || d.bus.f == nil {
		return 0, errors.New("i2c device is nil")
	}
	if d.addr == 0 || d.addr > 0x7F {
		return 0, fmt.Errorf("invalid i2c addr 0x%X", d.addr)
	}

	msgs := make([]msg, 0, 2)
	if len(w) > 0 {
		msgs = append(msgs, msg{addr: d.addr, flags: 0, len: uint16(len(w)), buf: uintptr(unsafe.Pointer(&w[0]))})
	}
	if len(r) > 0 {
		msgs = append(msgs, msg{addr: d.addr, flags: i2cMrd, len: uint16(len(r)), buf: uintptr(unsafe.Pointer(&r[0]))})
	}
	if len(msgs) == 0 {
		return 0, nil
	}

	data := rdwrData{msgs: uintptr(unsafe.Pointer(&msgs[0])), nmsgs: uint32(len(msgs))}
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, d.bus.f.Fd(), uintptr(i2cRdwr), uintptr(unsafe.Pointer(&data)))
	if errno != 0 {
		return 0, errno
	}
	if len(r) > 0 {
		return len(r), nil
	}
	return len(w), nil
}
