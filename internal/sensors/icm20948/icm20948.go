package icm20948

import (
	"fmt"
	"time"

	"stratux-ng/internal/i2c"
)

var sleep = time.Sleep

// Minimal ICM-20948 driver.
//
// Focus: probe + basic accel/gyro reads for roll/pitch bring-up.
// Register choices mirror upstream Stratux probing:
// - WHO_AM_I at 0x00 should return 0xEA.

const (
	addrDefault = 0x68

	regWhoAmI  = 0x00
	whoAmIVal  = 0xEA
	regBankSel = 0x7F

	// Bank 0.
	regPwrMgmt1   = 0x06
	bitReset      = 0x80
	regAccelXoutH = 0x2D // contiguous accel+gyro block
	regGyroXoutH  = 0x33
	regTempOutH   = 0x39
	regIntEnable  = 0x38
	regIntPinCfg  = 0x0F // per ICM20948 map, not used here

	// Bank 2.
	bank2           = 2
	regGyroSmplrt   = 0x00
	regGyroConfig   = 0x01
	regAccelSmplrt2 = 0x11
	regAccelConfig  = 0x14

	fsGyro250dps = 0x00
	fsAccel4g    = 0x02
)

type Sample struct {
	Time time.Time
	// Accel in G.
	Ax, Ay, Az float64
	// Gyro in deg/s.
	Gx, Gy, Gz float64
}

type Device struct {
	dev regIO

	curBank byte
	// scales based on configured full-scale.
	scaleAccel float64
	scaleGyro  float64
}

type regIO interface {
	ReadRegU8(reg byte) (byte, error)
	ReadReg(reg byte, dst []byte) error
	WriteReg(reg, value byte) error
}

func DefaultAddress() uint16 { return addrDefault }

func New(dev *i2c.Dev) (*Device, error) {
	if dev == nil {
		return nil, fmt.Errorf("icm20948: dev is nil")
	}
	return newWithIO(dev)
}

func newWithIO(dev regIO) (*Device, error) {
	if dev == nil {
		return nil, fmt.Errorf("icm20948: dev is nil")
	}
	d := &Device{dev: dev, curBank: 0xFF}

	who, err := d.dev.ReadRegU8(regWhoAmI)
	if err != nil {
		return nil, fmt.Errorf("icm20948: whoami read failed: %w", err)
	}
	if who != whoAmIVal {
		return nil, fmt.Errorf("icm20948: whoami=0x%02X want 0x%02X", who, whoAmIVal)
	}

	if err := d.init(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Device) init() error {
	// Bank 0.
	if err := d.setBank(0); err != nil {
		return err
	}

	// Disable interrupts (default, but be explicit).
	_ = d.dev.WriteReg(regIntEnable, 0x00)

	// Reset.
	if err := d.dev.WriteReg(regPwrMgmt1, bitReset); err != nil {
		return fmt.Errorf("icm20948: reset failed: %w", err)
	}
	sleep(100 * time.Millisecond)

	// Wake + PLL clock.
	// From ICM-20948 register map: CLKSEL[2:0] should be 1..5 for full gyro performance.
	if err := d.dev.WriteReg(regPwrMgmt1, 0x01); err != nil {
		return fmt.Errorf("icm20948: wake failed: %w", err)
	}
	sleep(10 * time.Millisecond)

	// Configure accel/gyro full-scale and sample rates.
	// Mirror Stratux defaults: gyro=250dps, accel=4g, updateFreq=50Hz.
	if err := d.setBank(bank2); err != nil {
		return err
	}

	// Sample rate divider. ICM base is 1125 Hz.
	// sampRate = 1125/(div+1). For 50Hz -> div ~ 21.
	div := byte(1125/50 - 1)
	_ = d.dev.WriteReg(regGyroSmplrt, div)
	_ = d.dev.WriteReg(regAccelSmplrt2, div)

	if err := d.dev.WriteReg(regGyroConfig, fsGyro250dps); err != nil {
		return fmt.Errorf("icm20948: gyro config failed: %w", err)
	}
	if err := d.dev.WriteReg(regAccelConfig, fsAccel4g); err != nil {
		return fmt.Errorf("icm20948: accel config failed: %w", err)
	}

	// Return to bank 0 for reads.
	if err := d.setBank(0); err != nil {
		return err
	}

	d.scaleAccel = 4.0 / 32768.0
	d.scaleGyro = 250.0 / 32768.0
	return nil
}

func (d *Device) setBank(bank byte) error {
	if d.curBank == bank {
		return nil
	}
	if err := d.dev.WriteReg(regBankSel, bank<<4); err != nil {
		return fmt.Errorf("icm20948: set bank %d failed: %w", bank, err)
	}
	d.curBank = bank
	return nil
}

func (d *Device) Read() (Sample, error) {
	if d == nil {
		return Sample{}, fmt.Errorf("icm20948: device is nil")
	}
	// Ensure bank 0.
	if err := d.setBank(0); err != nil {
		return Sample{}, err
	}

	buf := make([]byte, 12)
	if err := d.dev.ReadReg(regAccelXoutH, buf); err != nil {
		return Sample{}, fmt.Errorf("icm20948: read sensors failed: %w", err)
	}

	ax := int16(buf[0])<<8 | int16(buf[1])
	ay := int16(buf[2])<<8 | int16(buf[3])
	az := int16(buf[4])<<8 | int16(buf[5])
	gx := int16(buf[6])<<8 | int16(buf[7])
	gy := int16(buf[8])<<8 | int16(buf[9])
	gz := int16(buf[10])<<8 | int16(buf[11])

	return Sample{
		Time: time.Now(),
		Ax:   float64(ax) * d.scaleAccel,
		Ay:   float64(ay) * d.scaleAccel,
		Az:   float64(az) * d.scaleAccel,
		Gx:   float64(gx) * d.scaleGyro,
		Gy:   float64(gy) * d.scaleGyro,
		Gz:   float64(gz) * d.scaleGyro,
	}, nil
}
