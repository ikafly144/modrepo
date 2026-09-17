//go:build windows

package main

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	moduser32   = windows.NewLazySystemDLL("user32.dll")
	modcomctl32 = windows.NewLazySystemDLL("comctl32.dll")
	modgdi32    = windows.NewLazySystemDLL("gdi32.dll")

	procInitCommonControlsEx = modcomctl32.NewProc("InitCommonControlsEx")
	procRegisterClassExW     = moduser32.NewProc("RegisterClassExW")
	procCreateWindowExW      = moduser32.NewProc("CreateWindowExW")
	procDefWindowProcW       = moduser32.NewProc("DefWindowProcW")
	procShowWindow           = moduser32.NewProc("ShowWindow")
	procUpdateWindow         = moduser32.NewProc("UpdateWindow")
	procGetMessageW          = moduser32.NewProc("GetMessageW")
	procTranslateMessage     = moduser32.NewProc("TranslateMessage")
	procDispatchMessageW     = moduser32.NewProc("DispatchMessageW")
	procPostQuitMessage      = moduser32.NewProc("PostQuitMessage")
	procSendMessageW         = moduser32.NewProc("SendMessageW")
	procSetWindowTextW       = moduser32.NewProc("SetWindowTextW")
	procGetSystemMetrics     = moduser32.NewProc("GetSystemMetrics")
	procDestroyWindow        = moduser32.NewProc("DestroyWindow")
	procCreateFontW          = modgdi32.NewProc("CreateFontW")
	procPostMessageW         = moduser32.NewProc("PostMessageW")
	procMessageBoxW          = moduser32.NewProc("MessageBoxW")
)

const (
	ICC_PROGRESS_CLASS = 0x00000020
	PBM_SETPOS         = 0x0402
	PBM_SETRANGE32     = 0x0406
	WM_DESTROY         = 0x0002
	WM_CLOSE           = 0x0010
	WM_SETFONT         = 0x0030
	WS_POPUP           = 0x80000000
	WS_CAPTION         = 0x00C00000
	WS_SYSMENU         = 0x00080000
	WS_VISIBLE         = 0x10000000
	WS_CHILD           = 0x40000000
	SW_SHOW            = 5
	SM_CXSCREEN        = 0
	SM_CYSCREEN        = 1
	COLOR_BTNFACE      = 15

	MB_OK          = 0x00000000
	MB_ICONERROR   = 0x00000010
	MB_SYSTEMMODAL = 0x00001000
)

type INITCOMMONCONTROLSEX struct {
	DwSize uint32
	DwICC  uint32
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type MSG struct {
	Hwnd    windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

func ShowErrorDialog(title, message string) {
	tPtr, _ := windows.UTF16PtrFromString(title)
	mPtr, _ := windows.UTF16PtrFromString(message)
	_, _, _ = procMessageBoxW.Call(0, uintptr(unsafe.Pointer(mPtr)), uintptr(unsafe.Pointer(tPtr)), MB_OK|MB_ICONERROR|MB_SYSTEMMODAL)
}

type NativeProgressDialog struct {
	hwnd      windows.HWND
	hStatus   windows.HWND
	hProgress windows.HWND
	done      chan struct{}
}

func ShowProgressDialog(title, initialStatus string) *NativeProgressDialog {
	ready := make(chan *NativeProgressDialog)

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		var icex INITCOMMONCONTROLSEX
		icex.DwSize = uint32(unsafe.Sizeof(icex))
		icex.DwICC = ICC_PROGRESS_CLASS
		_, _, _ = procInitCommonControlsEx.Call(uintptr(unsafe.Pointer(&icex)))

		className, _ := windows.UTF16PtrFromString(fmt.Sprintf("MODREPO_Updater_%d", time.Now().UnixNano()))
		wndTitle, _ := windows.UTF16PtrFromString(title)

		wndProc := syscall.NewCallback(func(hwnd windows.HWND, msg uint32, wParam, lParam uintptr) uintptr {
			switch msg {
			case WM_CLOSE:
				_, _, _ = procDestroyWindow.Call(uintptr(hwnd))
				return 0
			case WM_DESTROY:
				_, _, _ = procPostQuitMessage.Call(0)
				return 0
			}
			r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
			return r
		})

		var wc WNDCLASSEX
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		wc.LpfnWndProc = wndProc
		wc.HbrBackground = windows.Handle(COLOR_BTNFACE + 1)
		wc.LpszClassName = className

		_, _, _ = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

		cx, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
		cy, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
		width := int32(440)
		height := int32(160)
		x := (int32(cx) - width) / 2
		y := (int32(cy) - height) / 2

		hwndVal, _, _ := procCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(className)),
			uintptr(unsafe.Pointer(wndTitle)),
			WS_POPUP|WS_CAPTION|WS_SYSMENU|WS_VISIBLE,
			uintptr(x), uintptr(y), uintptr(width), uintptr(height),
			0, 0, 0, 0,
		)
		if hwndVal == 0 {
			ready <- nil
			return
		}
		hwnd := windows.HWND(hwndVal)

		hFontVal, _, _ := procCreateFontW.Call(
			15, 0, 0, 0, 400, 0, 0, 0,
			1, 0, 0, 5, 0,
			uintptr(unsafe.Pointer(windows.StringToUTF16Ptr("Segoe UI"))),
		)
		hFont := uintptr(hFontVal)

		staticClass, _ := windows.UTF16PtrFromString("STATIC")
		progressClass, _ := windows.UTF16PtrFromString("msctls_progress32")

		titleText, _ := windows.UTF16PtrFromString("MODREPO を最新バージョンに更新しています...")
		hTitle, _, _ := procCreateWindowExW.Call(
			0, uintptr(unsafe.Pointer(staticClass)), uintptr(unsafe.Pointer(titleText)),
			WS_CHILD|WS_VISIBLE,
			25, 20, 390, 22,
			hwndVal, 101, 0, 0,
		)
		_, _, _ = procSendMessageW.Call(hTitle, WM_SETFONT, hFont, 1)

		statusText, _ := windows.UTF16PtrFromString(initialStatus)
		hStatusVal, _, _ := procCreateWindowExW.Call(
			0, uintptr(unsafe.Pointer(staticClass)), uintptr(unsafe.Pointer(statusText)),
			WS_CHILD|WS_VISIBLE,
			25, 46, 390, 20,
			hwndVal, 102, 0, 0,
		)
		_, _, _ = procSendMessageW.Call(hStatusVal, WM_SETFONT, hFont, 1)

		hProgressVal, _, _ := procCreateWindowExW.Call(
			0, uintptr(unsafe.Pointer(progressClass)), 0,
			WS_CHILD|WS_VISIBLE,
			25, 75, 380, 22,
			hwndVal, 103, 0, 0,
		)
		_, _, _ = procSendMessageW.Call(hProgressVal, PBM_SETRANGE32, 0, 100)

		_, _, _ = procShowWindow.Call(hwndVal, SW_SHOW)
		_, _, _ = procUpdateWindow.Call(hwndVal)

		npd := &NativeProgressDialog{
			hwnd:      hwnd,
			hStatus:   windows.HWND(hStatusVal),
			hProgress: windows.HWND(hProgressVal),
			done:      make(chan struct{}),
		}
		ready <- npd

		var msg MSG
		for {
			r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
			if int32(r) <= 0 {
				break
			}
			_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
		close(npd.done)
	}()

	return <-ready
}

func (d *NativeProgressDialog) Update(status string, pct int) {
	if d == nil || d.hwnd == 0 {
		return
	}
	if pct >= 0 {
		_, _, _ = procSendMessageW.Call(uintptr(d.hProgress), PBM_SETPOS, uintptr(pct), 0)
	}
	if status != "" {
		strPtr, _ := windows.UTF16PtrFromString(status)
		_, _, _ = procSetWindowTextW.Call(uintptr(d.hStatus), uintptr(unsafe.Pointer(strPtr)))
	}
}

func (d *NativeProgressDialog) Close() {
	if d == nil || d.hwnd == 0 {
		return
	}
	_, _, _ = procPostMessageW.Call(uintptr(d.hwnd), WM_CLOSE, 0, 0)
	<-d.done
}
