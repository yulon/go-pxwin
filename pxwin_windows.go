package pxwin

import (
	"syscall"
	"unsafe"
	"strings"
	//"fmt"
)

const (
	ws_visible = 0x10000000
	ws_caption = 0x00C00000
	ws_sysmenu = 0x00080000
	ws_overlapped = 0x00000000
	ws_thickframe = 0x00040000
	ws_maximizebox = 0x00010000
	ws_minimizebox = 0x00020000

	wm_paint = 0x000F
	wm_keydown = 0x0100
	wm_keyup = 0x0101
	wm_destroy = 0x0002
)

var (
	user32, _ = syscall.LoadDLL("user32.dll")
	kernel32, _ = syscall.LoadDLL("kernel32.dll")

	hProcess, _, _ = kernel32.MustFindProc("GetModuleHandleW").Call(0)

	defWindowProc = user32.MustFindProc("DefWindowProcW")
	postQuitMessage = user32.MustFindProc("PostQuitMessage")

	loadCursor = user32.MustFindProc("LoadCursorW")
	cArrow, _, _ = loadCursor.Call(0, 32512)

	winMap = map[uintptr]*windowsWindow{}

	wClass = &wndclassex{
		cbSize: 48,
		style: 2 | 1 | 8,
		hInstance: hProcess,
		hCursor: cArrow,
		hbrBackground: 15 + 1,
		lpszClassName: uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("pxwin"))),
		lpfnWndProc: syscall.NewCallback(func(hWnd, uMsg, wParam, lParam uintptr) uintptr {
			win, ok := winMap[hWnd]
			if ok {
				h, ok := win.msgHandlers[uMsg]
				if ok {
					println(1122)
					ret := h(wParam, lParam)
					if ret {
						return 0
					}
				}
			}
			if uMsg == wm_destroy {
				delete(winMap, hWnd)
				if len(winMap) == 0 {
					postQuitMessage.Call(0)
					return 0
				}
			}
			ret, _, _ := defWindowProc.Call(hWnd, uMsg, wParam, lParam)
			return ret
		}),
	}

	registerClassEx = user32.MustFindProc("RegisterClassExW")
	createWindowEx = user32.MustFindProc("CreateWindowExW")
)

func Init() {
	registerClassEx.Call(uintptr(unsafe.Pointer(wClass)))
}

type msg struct{
	hwnd uintptr
	message uintptr
	wParam uintptr
	lParam uintptr
	time uintptr
	x uintptr
	y uintptr
}

func EventDrive() {
	GetMessage := user32.MustFindProc("GetMessageW")
	DispatchMessage := user32.MustFindProc("DispatchMessageW")
	TranslateMessage := user32.MustFindProc("TranslateMessage")
	msg := &msg{}

	for {
		ret, _, _ := GetMessage.Call(uintptr(unsafe.Pointer(msg)), 0, 0, 0)
		if ret == 0 {
			return
		}

		TranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
		DispatchMessage.Call(uintptr(unsafe.Pointer(msg)))
	}
}

type wndclassex struct{
	cbSize uintptr
	style uintptr
	lpfnWndProc uintptr
	cbClsExtra uintptr
	cbWndExtra uintptr
	hInstance uintptr
	hIcon uintptr
	hCursor uintptr
	hbrBackground uintptr
	lpszMenuName uintptr
	lpszClassName uintptr
	hIconSm uintptr
}

type windowsWindow struct{
	hWnd uintptr
	msgHandlers map[uintptr]msgHandler
}

type msgHandler func(wParam, lParam uintptr) bool

var handlerConv = map[int]func(eh EventHandler) (msg uintptr, mh msgHandler) {
	EventPaint: func(eh EventHandler) (msg uintptr, mh msgHandler) {
		return wm_paint, func(wParam, lParam uintptr) bool {
			eh(0)
			return true
		}
	},
	EventKeyDown: func(eh EventHandler) (msg uintptr, mh msgHandler) {
		return wm_keydown, func(wParam, lParam uintptr) bool {
			eh(int(wParam))
			return false
		}
	},
	EventKeyUp: func(eh EventHandler) (msg uintptr, mh msgHandler) {
		return wm_keyup, func(wParam, lParam uintptr) bool {
			eh(int(wParam))
			return false
		}
	},
}

func (w *windowsWindow) ListenEvent(event int, eh EventHandler) {
	msg, mh := handlerConv[event](eh)
	w.msgHandlers[msg] = mh
}

var (
	getWindowTextLength = user32.MustFindProc("GetWindowTextLengthW")
	getWindowText = user32.MustFindProc("GetWindowTextW")
)

func (w *windowsWindow) GetTitle() string {
	leng, _, _ := getWindowTextLength.Call(w.hWnd)
	str := syscall.StringToUTF16(strings.Repeat(" ", int(leng)))
	getWindowText.Call(w.hWnd, uintptr(unsafe.Pointer(&str[0])), leng)
	return syscall.UTF16ToString(str)
}

var setWindowText = user32.MustFindProc("SetWindowTextW")

func (w *windowsWindow) SetTitle(title string) {
	setWindowText.Call(w.hWnd, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))))
}

var getWindowRect = user32.MustFindProc("GetWindowRect")

type windowsRect struct{
	Left int
	Top int
	Right int
	Bottom int
}

func (w *windowsWindow) GetRect() *Rect {
	wr := &windowsRect{}
	getWindowRect.Call(w.hWnd, uintptr(unsafe.Pointer(wr)))
	return &Rect{
		wr.Left,
		wr.Top,
		wr.Right - wr.Left,
		wr.Bottom - wr.Top,
	}
}

func New() Window {
	hWnd, _, _ := createWindowEx.Call(
		1,
		wClass.lpszClassName,
		0,
		ws_visible | ws_caption | ws_sysmenu | ws_overlapped | ws_thickframe | ws_maximizebox | ws_minimizebox,
		0,
		0,
		500,
		500,
		0,
		0,
		hProcess,
		0,
	)
	win := &windowsWindow{hWnd, map[uintptr]msgHandler{}}
	winMap[hWnd] = win
	return win
}
