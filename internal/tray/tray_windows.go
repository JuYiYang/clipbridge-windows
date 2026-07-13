//go:build windows

package tray

import (
	"context"
	"syscall"
	"unsafe"
)

type Options struct {
	SettingsURL string
	OnSyncNow   func()
	OnQuit      func()
}

const (
	wmDestroy       = 0x0002
	wmCommand       = 0x0111
	wmClose         = 0x0010
	wmUser          = 0x0400
	wmTray          = wmUser + 1
	wmLButtonDblClk = 0x0203
	wmRButtonUp     = 0x0205

	nimAdd     = 0x00000000
	nimDelete  = 0x00000002
	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004

	swShow = 5

	idiApplication = 32512

	menuSettings = 1001
	menuSyncNow  = 1002
	menuQuit     = 1003
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	shell32              = syscall.NewLazyDLL("shell32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procRegisterClassEx  = user32.NewProc("RegisterClassExW")
	procCreateWindowEx   = user32.NewProc("CreateWindowExW")
	procDefWindowProc    = user32.NewProc("DefWindowProcW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procGetMessage       = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")
	procLoadIcon         = user32.NewProc("LoadIconW")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procAppendMenu       = user32.NewProc("AppendMenuW")
	procTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	procSetForeground    = user32.NewProc("SetForegroundWindow")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procPostMessage      = user32.NewProc("PostMessageW")
	procGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")
	procShellNotifyIcon  = shell32.NewProc("Shell_NotifyIconW")
	procShellExecute     = shell32.NewProc("ShellExecuteW")
)

type point struct {
	X int32
	Y int32
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type notifyIconData struct {
	Size             uint32
	HWnd             uintptr
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	TimeoutOrVersion uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	GuidItem         [16]byte
	BalloonIcon      uintptr
}

type runtimeState struct {
	options Options
	hwnd    uintptr
	menu    uintptr
	nid     notifyIconData
}

var current *runtimeState

func Run(ctx context.Context, options Options) error {
	className := syscall.StringToUTF16Ptr("ClipBridgeTrayWindow")
	instance, _, _ := procGetModuleHandle.Call(0)
	windowProc := syscall.NewCallback(wndProc)

	class := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   windowProc,
		Instance:  instance,
		ClassName: className,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&class)))

	hwnd, _, err := procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("ClipBridge"))),
		0,
		0, 0, 0, 0,
		0, 0, instance, 0,
	)
	if hwnd == 0 {
		return err
	}

	menu := createMenu()
	icon, _, _ := procLoadIcon.Call(0, uintptr(idiApplication))
	state := &runtimeState{
		options: options,
		hwnd:    hwnd,
		menu:    menu,
		nid: notifyIconData{
			Size:            uint32(unsafe.Sizeof(notifyIconData{})),
			HWnd:            hwnd,
			ID:              1,
			Flags:           nifMessage | nifIcon | nifTip,
			CallbackMessage: wmTray,
			Icon:            icon,
		},
	}
	copy(state.nid.Tip[:], syscall.StringToUTF16("ClipBridge"))
	current = state
	procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&state.nid)))
	defer procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&state.nid)))

	go func() {
		<-ctx.Done()
		procPostMessage.Call(hwnd, wmClose, 0, 0)
	}()

	var message msg
	for {
		ret, _, callErr := procGetMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(ret) == -1 {
			return callErr
		}
		if ret == 0 {
			return nil
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func createMenu() uintptr {
	menu, _, _ := procCreatePopupMenu.Call()
	appendMenu(menu, menuSettings, "设置...")
	appendMenu(menu, menuSyncNow, "立即同步")
	appendMenu(menu, menuQuit, "退出")
	return menu
}

func appendMenu(menu uintptr, id uintptr, text string) {
	const mfString = 0x00000000
	procAppendMenu.Call(menu, mfString, id, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))))
}

func wndProc(hwnd uintptr, message uint32, wParam uintptr, lParam uintptr) uintptr {
	switch message {
	case wmTray:
		switch lParam {
		case wmLButtonDblClk:
			openSettings()
			return 0
		case wmRButtonUp:
			showMenu(hwnd)
			return 0
		}
	case wmCommand:
		switch wParam & 0xffff {
		case menuSettings:
			openSettings()
		case menuSyncNow:
			if current != nil && current.options.OnSyncNow != nil {
				current.options.OnSyncNow()
			}
		case menuQuit:
			if current != nil && current.options.OnQuit != nil {
				current.options.OnQuit()
			}
			procDestroyWindow.Call(hwnd)
		}
		return 0
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := procDefWindowProc.Call(hwnd, uintptr(message), wParam, lParam)
	return ret
}

func showMenu(hwnd uintptr) {
	const tpmRightButton = 0x0002
	var cursor point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&cursor)))
	procSetForeground.Call(hwnd)
	if current != nil {
		procTrackPopupMenu.Call(current.menu, tpmRightButton, uintptr(cursor.X), uintptr(cursor.Y), 0, hwnd, 0)
	}
}

func openSettings() {
	if current == nil || current.options.SettingsURL == "" {
		return
	}
	procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("open"))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(current.options.SettingsURL))),
		0,
		0,
		swShow,
	)
}
