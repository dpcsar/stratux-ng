//go:build linux

package gps

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func openSerial(path string, baud int) (*os.File, error) {
	flag := unix.O_RDWR | unix.O_NOCTTY
	fd, err := unix.Open(path, flag, 0)
	if err != nil {
		return nil, err
	}

	// Best-effort: if anything below fails, close fd.
	ok := false
	defer func() {
		if !ok {
			_ = unix.Close(fd)
		}
	}()

	t, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}

	spd, err := baudToUnix(baud)
	if err != nil {
		return nil, err
	}

	// Raw-ish mode (minimal line processing) for NMEA.
	t.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	t.Oflag &^= unix.OPOST
	t.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	t.Cflag &^= unix.CSIZE | unix.PARENB
	t.Cflag |= unix.CS8

	// 1 second read timeout, return as soon as at least 1 byte is available.
	t.Cc[unix.VMIN] = 1
	t.Cc[unix.VTIME] = 10

	// Set baud.
	t.Cflag &^= unix.CBAUD
	t.Cflag |= spd
	t.Ispeed = spd
	t.Ospeed = spd

	if err := unix.IoctlSetTermios(fd, unix.TCSETS, t); err != nil {
		return nil, err
	}

	f := os.NewFile(uintptr(fd), path)
	if f == nil {
		return nil, fmt.Errorf("os.NewFile failed")
	}
	ok = true
	return f, nil
}

func baudToUnix(baud int) (uint32, error) {
	switch baud {
	case 4800:
		return unix.B4800, nil
	case 9600:
		return unix.B9600, nil
	case 19200:
		return unix.B19200, nil
	case 38400:
		return unix.B38400, nil
	case 57600:
		return unix.B57600, nil
	case 115200:
		return unix.B115200, nil
	default:
		return 0, fmt.Errorf("unsupported baud %d", baud)
	}
}
