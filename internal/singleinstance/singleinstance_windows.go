//go:build windows

package singleinstance

import (
	"errors"
	"syscall"
	"unsafe"
)

const errorAlreadyExists = 183

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW = kernel32.NewProc("CreateMutexW")
	procCloseHandle  = kernel32.NewProc("CloseHandle")

	ErrAlreadyRunning = errors.New("clipbridge is already running")
)

type Lock struct {
	handle uintptr
}

func Acquire() (*Lock, error) {
	name, err := syscall.UTF16PtrFromString(`Global\ClipBridgeWindows`)
	if err != nil {
		return nil, err
	}

	handle, _, callErr := procCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		if callErr != syscall.Errno(0) {
			return nil, callErr
		}
		return nil, errors.New("create single instance mutex failed")
	}

	if errno, ok := callErr.(syscall.Errno); ok && errno == errorAlreadyExists {
		procCloseHandle.Call(handle)
		return nil, ErrAlreadyRunning
	}

	return &Lock{handle: handle}, nil
}

func (l *Lock) Release() {
	if l == nil || l.handle == 0 {
		return
	}

	procCloseHandle.Call(l.handle)
	l.handle = 0
}
