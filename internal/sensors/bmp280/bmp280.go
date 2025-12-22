package bmp280

import (
	"encoding/binary"
	"fmt"
	"time"

	"stratux-ng/internal/i2c"
)

var sleep = time.Sleep

// Minimal BMP280 driver.
//
// Supports chip ID + reading compensated temperature/pressure.

const (
	addrDefault = 0x77

	regID        = 0xD0
	chipIDBMP280 = 0x58

	regReset = 0xE0
	resetCmd = 0xB6

	regCalib00 = 0x88
	calibLen   = 24

	regCtrlMeas = 0xF4
	regConfig   = 0xF5
	regPressMsb = 0xF7
)

type Device struct {
	dev regIO

	// Calibration.
	digT1 uint16
	digT2 int16
	digT3 int16
	digP1 uint16
	digP2 int16
	digP3 int16
	digP4 int16
	digP5 int16
	digP6 int16
	digP7 int16
	digP8 int16
	digP9 int16

	tFine int32
}

type regIO interface {
	ReadRegU8(reg byte) (byte, error)
	ReadReg(reg byte, dst []byte) error
	WriteReg(reg, value byte) error
}

func DefaultAddress() uint16 { return addrDefault }

func New(dev *i2c.Dev) (*Device, error) {
	if dev == nil {
		return nil, fmt.Errorf("bmp280: dev is nil")
	}
	return newWithIO(dev)
}

func newWithIO(dev regIO) (*Device, error) {
	if dev == nil {
		return nil, fmt.Errorf("bmp280: dev is nil")
	}
	d := &Device{dev: dev}

	id, err := d.dev.ReadRegU8(regID)
	if err != nil {
		return nil, fmt.Errorf("bmp280: id read failed: %w", err)
	}
	if id != chipIDBMP280 {
		return nil, fmt.Errorf("bmp280: chip id=0x%02X want 0x%02X", id, chipIDBMP280)
	}

	// Soft reset (optional but makes config consistent).
	// Datasheet: after reset, the NVM calibration coefficients are copied and
	// may take a couple milliseconds. If we read too early we can get zeros and
	// end up with compensated pressure=0.
	_ = d.dev.WriteReg(regReset, resetCmd)
	sleep(5 * time.Millisecond)

	// Read calibration with a couple of retries to avoid transient zero reads.
	var calibErr error
	for i := 0; i < 3; i++ {
		calibErr = d.readCalibration()
		if calibErr != nil {
			sleep(5 * time.Millisecond)
			continue
		}
		// Basic sanity: these are never expected to be 0 on a real BMP280.
		if d.digT1 != 0 && d.digP1 != 0 {
			calibErr = nil
			break
		}
		calibErr = fmt.Errorf("bmp280: calibration invalid (digT1=%d digP1=%d)", d.digT1, d.digP1)
		sleep(5 * time.Millisecond)
	}
	if calibErr != nil {
		return nil, calibErr
	}

	// Config:
	// - standby 0.5ms (t_sb=000)
	// - IIR filter off (filter=000)
	// - spi3w_en=0
	_ = d.dev.WriteReg(regConfig, 0x00)

	// ctrl_meas:
	// osrs_t = x2 (010)
	// osrs_p = x16 (101)
	// mode = normal (11)
	ctrl := byte(0x02<<5) | byte(0x05<<2) | 0x03
	if err := d.dev.WriteReg(regCtrlMeas, ctrl); err != nil {
		return nil, fmt.Errorf("bmp280: ctrl_meas write failed: %w", err)
	}

	return d, nil
}

func (d *Device) readCalibration() error {
	buf := make([]byte, calibLen)
	if err := d.dev.ReadReg(regCalib00, buf); err != nil {
		return fmt.Errorf("bmp280: read calib failed: %w", err)
	}
	// Little endian.
	d.digT1 = binary.LittleEndian.Uint16(buf[0:2])
	d.digT2 = int16(binary.LittleEndian.Uint16(buf[2:4]))
	d.digT3 = int16(binary.LittleEndian.Uint16(buf[4:6]))
	d.digP1 = binary.LittleEndian.Uint16(buf[6:8])
	d.digP2 = int16(binary.LittleEndian.Uint16(buf[8:10]))
	d.digP3 = int16(binary.LittleEndian.Uint16(buf[10:12]))
	d.digP4 = int16(binary.LittleEndian.Uint16(buf[12:14]))
	d.digP5 = int16(binary.LittleEndian.Uint16(buf[14:16]))
	d.digP6 = int16(binary.LittleEndian.Uint16(buf[16:18]))
	d.digP7 = int16(binary.LittleEndian.Uint16(buf[18:20]))
	d.digP8 = int16(binary.LittleEndian.Uint16(buf[20:22]))
	d.digP9 = int16(binary.LittleEndian.Uint16(buf[22:24]))
	return nil
}

// Read returns compensated temperature (C) and pressure (Pa).
func (d *Device) Read() (tempC float64, pressPa float64, err error) {
	buf := make([]byte, 6)
	if err := d.dev.ReadReg(regPressMsb, buf); err != nil {
		return 0, 0, fmt.Errorf("bmp280: read data failed: %w", err)
	}

	adcP := int32(buf[0])<<12 | int32(buf[1])<<4 | int32(buf[2])>>4
	adcT := int32(buf[3])<<12 | int32(buf[4])<<4 | int32(buf[5])>>4

	tFine, t := d.compensateTemp(adcT)
	d.tFine = tFine
	p := d.compensatePress(adcP)

	return t, p, nil
}

func (d *Device) compensateTemp(adcT int32) (tFine int32, tempC float64) {
	var1 := (float64(adcT)/16384.0 - float64(d.digT1)/1024.0) * float64(d.digT2)
	var2 := (float64(adcT)/131072.0 - float64(d.digT1)/8192.0)
	var2 = var2 * var2 * float64(d.digT3)
	tFineF := var1 + var2
	tFine = int32(tFineF)
	tempC = tFineF / 5120.0
	return tFine, tempC
}

func (d *Device) compensatePress(adcP int32) float64 {
	// Datasheet algorithm, using float64 for simplicity.
	var1 := float64(d.tFine)/2.0 - 64000.0
	var2 := var1 * var1 * float64(d.digP6) / 32768.0
	var2 = var2 + var1*float64(d.digP5)*2.0
	var2 = var2/4.0 + float64(d.digP4)*65536.0
	var1 = (float64(d.digP3)*var1*var1/524288.0 + float64(d.digP2)*var1) / 524288.0
	var1 = (1.0 + var1/32768.0) * float64(d.digP1)
	if var1 == 0 {
		return 0
	}
	p := 1048576.0 - float64(adcP)
	p = (p - var2/4096.0) * 6250.0 / var1
	var1 = float64(d.digP9) * p * p / 2147483648.0
	var2 = p * float64(d.digP8) / 32768.0
	p = p + (var1+var2+float64(d.digP7))/16.0
	return p
}
