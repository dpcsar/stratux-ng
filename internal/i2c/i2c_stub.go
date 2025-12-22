//go:build !linux

package i2c

import "fmt"

type Bus struct{}

type Dev struct{}

func Open(path string) (*Bus, error) { return nil, fmt.Errorf("i2c: unsupported OS (need linux)") }

func (b *Bus) Close() error { return nil }

func (b *Bus) Dev(addr uint16) *Dev { return nil }

func (d *Dev) Write(p []byte) error               { return fmt.Errorf("i2c: unsupported OS") }
func (d *Dev) Read(p []byte) error                { return fmt.Errorf("i2c: unsupported OS") }
func (d *Dev) WriteRead(w, r []byte) error        { return fmt.Errorf("i2c: unsupported OS") }
func (d *Dev) ReadReg(reg byte, dst []byte) error { return fmt.Errorf("i2c: unsupported OS") }
func (d *Dev) ReadRegU8(reg byte) (byte, error)   { return 0, fmt.Errorf("i2c: unsupported OS") }
func (d *Dev) WriteReg(reg, value byte) error     { return fmt.Errorf("i2c: unsupported OS") }
