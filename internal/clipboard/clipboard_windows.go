//go:build windows

package clipboard

import (
	"context"
	"errors"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const cfUnicodeText = 13

var (
	user32                    = syscall.NewLazyDLL("user32.dll")
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetClipboardSeqNumber = user32.NewProc("GetClipboardSequenceNumber")
	procIsClipboardAvailable  = user32.NewProc("IsClipboardFormatAvailable")
	procOpenClipboard         = user32.NewProc("OpenClipboard")
	procCloseClipboard        = user32.NewProc("CloseClipboard")
	procGetClipboardData      = user32.NewProc("GetClipboardData")
	procGlobalLock            = kernel32.NewProc("GlobalLock")
	procGlobalUnlock          = kernel32.NewProc("GlobalUnlock")
)

type PollingWatcher struct {
	interval time.Duration
	events   chan Event
}

func NewPollingWatcher(interval time.Duration) *PollingWatcher {
	return &PollingWatcher{
		interval: interval,
		events:   make(chan Event, 8),
	}
}

func (w *PollingWatcher) Events() <-chan Event {
	return w.events
}

func (w *PollingWatcher) Run(ctx context.Context) error {
	defer close(w.events)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	var lastSequence uintptr
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			sequence := clipboardSequenceNumber()
			if sequence == 0 || sequence == lastSequence {
				continue
			}
			lastSequence = sequence

			text, err := readUnicodeText()
			if err != nil || text == "" {
				continue
			}

			select {
			case w.events <- Event{Text: text, Captured: time.Now().UTC()}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func clipboardSequenceNumber() uintptr {
	value, _, _ := procGetClipboardSeqNumber.Call()
	return value
}

func readUnicodeText() (string, error) {
	available, _, _ := procIsClipboardAvailable.Call(uintptr(cfUnicodeText))
	if available == 0 {
		return "", nil
	}

	opened, _, err := procOpenClipboard.Call(0)
	if opened == 0 {
		if err != syscall.Errno(0) {
			return "", err
		}
		return "", errors.New("open clipboard failed")
	}
	defer procCloseClipboard.Call()

	handle, _, err := procGetClipboardData.Call(uintptr(cfUnicodeText))
	if handle == 0 {
		if err != syscall.Errno(0) {
			return "", err
		}
		return "", errors.New("get clipboard data failed")
	}

	ptr, _, err := procGlobalLock.Call(handle)
	if ptr == 0 {
		if err != syscall.Errno(0) {
			return "", err
		}
		return "", errors.New("lock clipboard data failed")
	}
	defer procGlobalUnlock.Call(handle)

	return utf16PtrToString((*uint16)(unsafe.Pointer(ptr))), nil
}

func utf16PtrToString(ptr *uint16) string {
	if ptr == nil {
		return ""
	}
	var data []uint16
	for offset := uintptr(0); ; offset += unsafe.Sizeof(*ptr) {
		value := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + offset))
		if value == 0 {
			break
		}
		data = append(data, value)
	}
	return string(utf16.Decode(data))
}
