package libconpty

import (
	"fmt"
	. "github.com/bgrewell/go-conpty/libconpty/types"
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

const (
	S_OK                                        = 0x00000000
	PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE uintptr = 0x20016
)

var (
	win32                              = windows.NewLazySystemDLL("kernel32.dll")
	fCreatePseudoConsole               = win32.NewProc("CreatePseudoConsole")
	fResizePseudoConsole               = win32.NewProc("ResizePseudoConsole")
	fClosePseudoConsole                = win32.NewProc("ClosePseudoConsole")
	fInitializeProcThreadAttributeList = win32.NewProc("InitializeProcThreadAttributeList")
	fUpdateProcThreadAttribute         = win32.NewProc("UpdateProcThreadAttribute")
	fPeekNamedPipe					   = win32.NewProc("PeekNamedPipe")
	fGetProcessHeap					   = win32.NewProc("GetProcessHeap")
	fHeapAlloc						   = win32.NewProc("HeapAlloc")
)

type ConPty struct {
	hPC         HPCON `json:"handle_pseudoconsole"` // Handle to pseudo console device
	ptyIn       windows.Handle
	ptyOut      windows.Handle
	cmdIn       windows.Handle
	cmdOut      windows.Handle
	cmd         string
	consoleSize COORD
	pi          *windows.ProcessInformation
}

func NewConPty(cmd string, cols int16, rows int16) (conpty *ConPty, err error) {
	conpty = &ConPty{
		hPC:    0,
		ptyIn:  0,
		ptyOut: 0,
		cmdIn:  0,
		cmdOut: 0,
		cmd:    cmd,
		consoleSize: COORD{
			X: cols,
			Y: rows,
		},
	}
	err = conpty.Initialize()
	return conpty, err
}

func (c *ConPty) Initialize() (err error) {
	// Create pipes
	if err = c.setupPipes(); err != nil {
		return err
	}

	// Create pseudo console
	if err = c.createPseudoConsole(); err != nil {
		return err
	}

	// Close the handles (they are already dup'ed into the ConHost)
	if err = c.closePtyHandles(); err != nil {
		return err
	}

	// Initialize startup info
	var si *StartupInfoEx
	if si, err = c.initStartupInfoEx(); err != nil {
		return err
	}

	// Create process
	return c.createProcess(si)

}

func (c *ConPty) createProcess(si *StartupInfoEx) error {
	cmdline, err := windows.UTF16PtrFromString(c.cmd)
	if err != nil {
		return err
	}
	c.pi = new(windows.ProcessInformation)
	err = windows.CreateProcess(
		nil,                                  	// _in_opt_			LPCTSTR
		cmdline,                              			// _Inout_opt_		LPTSTR
		nil,                                  // _In_opt_			LPSECURITY_ATTRIBUTES
		nil,                                 // _In_opt_		LPSECURITY_ATTRIBUTES
		false,                               // _In_			BOOL
		windows.EXTENDED_STARTUPINFO_PRESENT, 			// _In_ 			DWORD
		nil,                                  		// _In_opt_			LPVOID
		nil,                                  	// _In_opt_			LPCTSTR
		&si.StartupInfo,                      			// _In_				LPSTARTUPINFO
		c.pi,                                 			// _Out_
	)
	if err != nil {
		return err
	}

	//event, err := windows.WaitForSingleObject(c.pi.Thread, 500)
	_, err = windows.WaitForSingleObject(c.pi.Thread, 5000)
	if err != nil {
		return err
	}
	//if event != 0x0 {
	//	fmt.Println("WaitForSingleObject returned unexpected value: %d", event)
	//}
	return nil
}

func (c *ConPty) initStartupInfoEx() (si *StartupInfoEx, err error) {
	if fInitializeProcThreadAttributeList.Find() != nil {
		return nil, fmt.Errorf("Unsupported version of Windows. InitializeProcThreadAttributeList not found")
	}
	if fUpdateProcThreadAttribute.Find() != nil {
		return nil, fmt.Errorf("Unsupported version of Windows. UpdateProcThreadAttribute not found")
	}
	si = &StartupInfoEx{}
	si.StartupInfo.Cb = uint32(unsafe.Sizeof(StartupInfoEx{}))
	lpSize := uint32(0)
	fInitializeProcThreadAttributeList.Call(0, 1, 0, uintptr(unsafe.Pointer(&lpSize)))

	heap, _, err := fGetProcessHeap.Call()
	if err != syscall.Errno(0) {
		return nil, fmt.Errorf("Failed to get process heap: %v", err)
	}
	const heapZeroMem = 0x00000008
	si.AttributeList, _, err = fHeapAlloc.Call(heap, heapZeroMem, uintptr(lpSize))
	if err != syscall.Errno(0) {
		return nil, fmt.Errorf("Failed to allocate space on the heap: %v", err)
	}
	ret, _, err := fInitializeProcThreadAttributeList.Call(uintptr(unsafe.Pointer(si.AttributeList)), 1, 0, uintptr(unsafe.Pointer(&lpSize)))
	if ret != 1 {
		return nil, fmt.Errorf("Failed to initialize thread attribute list: %v", err)
	}
	ret, _, err = fUpdateProcThreadAttribute.Call(uintptr(unsafe.Pointer(si.AttributeList)),
		0,
		PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		uintptr(c.hPC),
		unsafe.Sizeof(c.hPC),
		0,
		0)
	if ret != 1 {
		return nil, fmt.Errorf("Failed to update thread attribute list: %v", err)
	}

	return si, nil
}

func (c *ConPty) closePtyHandles() error {
	if err := windows.CloseHandle(c.ptyIn); err != nil {
		return err
	}
	if err := windows.CloseHandle(c.ptyOut); err != nil {
		return err
	}
	return nil
}

func (c *ConPty) createPseudoConsole() error {
	if fCreatePseudoConsole.Find() != nil {
		return fmt.Errorf("Unsupported version of Windows. CreatePseudoConsole not found")
	}
	ret, _, _ := fCreatePseudoConsole.Call(
		c.consoleSize.Pack(),
		uintptr(c.ptyIn),
		uintptr(c.ptyOut),
		0,
		uintptr(unsafe.Pointer(&c.hPC)))
	if ret != S_OK {
		return fmt.Errorf("CreatePseudoConsole() failed with status 0x%x", ret)
	}
	return nil
}

func (c *ConPty) setupPipes() error {
	if err := windows.CreatePipe(&c.ptyIn, &c.cmdIn, nil, 0); err != nil {
		return err
	}
	if err := windows.CreatePipe(&c.cmdOut, &c.ptyOut, nil, 0); err != nil {
		return err
	}
	return nil
}

func (c *ConPty) dataAvailable() (bytesAvailable int, err error) {
	if fPeekNamedPipe.Find() != nil {
		return -1, fmt.Errorf("Unsupported version of Windows. PeekNamedPipe not found")
	}
	maxRead := 1024 * 8
	numRead := uint32(0)
	numAvail := uint32(0)
	numLeft := uint32(0)
	buf := make([]byte, maxRead)
	_p0 := &buf[0]

	ret, _, err := fPeekNamedPipe.Call(uintptr(c.cmdOut),
									   uintptr(unsafe.Pointer(_p0)),
									   uintptr(len(buf)),
									   uintptr(unsafe.Pointer(&numRead)),
									   uintptr(unsafe.Pointer(&numAvail)),
									   uintptr(unsafe.Pointer(&numLeft)))
	if err != nil || ret == 0x0 {
		return -1, err
	}
	return int(numAvail), nil
}

func (c *ConPty) Read(p []byte) (n int, err error) {
	if avail, err := c.dataAvailable(); avail <= 0 || err != nil {
		if err != nil && err != syscall.Errno(0) {
			return 0, err
		}
		return 0, nil
	}
	numRead := uint32(0)
	err = windows.ReadFile(c.cmdOut, p, &numRead, nil)
	if err != nil && err != syscall.Errno(0) {
		return 0, err
	}
	return int(numRead), nil
}

func (c *ConPty) Write(p []byte) (n int, err error) {
	numWritten := uint32(0)
	err = windows.WriteFile(c.cmdIn, p, &numWritten, nil)
	if err != nil && err != syscall.Errno(0) {
		return 0, err
	}
	return int(numWritten), nil
}

func (c *ConPty) Close() {
	windows.CloseHandle(c.cmdIn)
	windows.CloseHandle(c.cmdOut)
	// todo: need to implement DeleteProcThreadAttributeList to cleanup if garbage collector doesn't get it
	// fDeleteProcThreadAttributeList(c.si.lpAttributeList)
	fClosePseudoConsole.Call(uintptr(c.hPC))
	windows.CloseHandle(c.pi.Thread)
	windows.CloseHandle(c.pi.Process)
}
