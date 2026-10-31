//go:build windows

package terminal

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

const enableVirtualTerminalProcessing = 0x0004

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	getConsoleMode             = kernel32.NewProc("GetConsoleMode")
	setConsoleMode             = kernel32.NewProc("SetConsoleMode")
	getConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

// Prepare enables ANSI cursor movement on legacy Windows console hosts.
// Without this flag escape sequences may be printed as ordinary output,
// causing each rendered frame to append new lines.
func Prepare(out io.Writer) (func(), error) {
	f, ok := out.(*os.File)
	if !ok {
		return func() {}, nil
	}
	handle := f.Fd()
	var original uint32
	result, _, callErr := getConsoleMode.Call(handle, uintptr(unsafe.Pointer(&original)))
	if result == 0 {
		return nil, fmt.Errorf("stdout is not an interactive ANSI console: %v", callErr)
	}
	result, _, callErr = setConsoleMode.Call(handle, uintptr(original|enableVirtualTerminalProcessing))
	if result == 0 {
		return nil, fmt.Errorf("cannot enable Windows virtual terminal mode: %v", callErr)
	}
	return func() { _, _, _ = setConsoleMode.Call(handle, uintptr(original)) }, nil
}

type coord struct{ X, Y int16 }
type smallRect struct{ Left, Top, Right, Bottom int16 }
type info struct {
	Size, Cursor      coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

func Size() (int, int) {
	var v info
	ok, _, _ := getConsoleScreenBufferInfo.Call(os.Stdout.Fd(), uintptr(unsafe.Pointer(&v)))
	if ok != 0 {
		return int(v.Window.Right - v.Window.Left + 1), int(v.Window.Bottom - v.Window.Top + 1)
	}
	return fallback()
}
func fallback() (int, int) {
	w, h := 80, 24
	if v, e := strconv.Atoi(os.Getenv("COLUMNS")); e == nil && v > 0 {
		w = v
	}
	if v, e := strconv.Atoi(os.Getenv("LINES")); e == nil && v > 0 {
		h = v
	}
	return w, h
}
