//go:build !windows

package terminal

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

func Prepare(out io.Writer) (func(), error) {
	f, ok := out.(*os.File)
	if !ok {
		return func() {}, nil
	}
	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect terminal: %w", err)
	}
	if stat.Mode()&os.ModeCharDevice == 0 {
		return nil, fmt.Errorf("stdout is not an interactive terminal")
	}
	if os.Getenv("TERM") == "dumb" {
		return nil, fmt.Errorf("terminal does not support ANSI screen updates (TERM=dumb)")
	}
	return func() {}, nil
}

type winsize struct{ Row, Col, Xpixel, Ypixel uint16 }

func Size() (int, int) {
	ws := winsize{}
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, os.Stdout.Fd(), uintptr(0x5413), uintptr(unsafe.Pointer(&ws)))
	if e == 0 && ws.Col > 0 {
		return int(ws.Col), int(ws.Row)
	}
	w, h := 80, 24
	if v, x := strconv.Atoi(os.Getenv("COLUMNS")); x == nil && v > 0 {
		w = v
	}
	if v, x := strconv.Atoi(os.Getenv("LINES")); x == nil && v > 0 {
		h = v
	}
	return w, h
}
