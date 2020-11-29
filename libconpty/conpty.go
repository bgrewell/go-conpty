package libconpty

import (
	"fmt"
	. "github.com/bgrewell/go-conpty/libconpty/types"
	"golang.org/x/sys/windows"
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
		hPC:    -1,
		ptyIn:  -1,
		ptyOut: -1,
		cmdIn:  -1,
		cmdOut: -1,
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
	// Setup handles
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
	return c.createProcess(err, si)

}

func (c *ConPty) createProcess(err error, si *StartupInfoEx) error {
	cmdline, err := windows.UTF16PtrFromString(c.cmd)
	if err != nil {
		return err
	}
	c.pi = &windows.ProcessInformation{}
	err = windows.CreateProcess(
		nil,                                  	// _in_opt_			LPCTSTR
		cmdline,                              			// _Inout_opt_		LPTSTR
		nil,                                  // _In_opt_			LPSECURITY_ATTRIBUTES
		nil,                                 // _In_opt_			LPSECURITY_ATTRIBUTES
		false,                               // _In_				BOOL
		windows.EXTENDED_STARTUPINFO_PRESENT, 			// _In_ 			DWORD
		nil,                                  		// _In_opt_			LPVOID
		nil,                                  	// _In_opt_			LPCTSTR
		&si.StartupInfo,                      			// _In_				LPSTARTUPINFO
		c.pi,                                 			// _Out_
	)
	if err != nil {
		return err
	}

	event, err := windows.WaitForSingleObject(c.pi.Thread, 500)
	if err != nil {
		return err
	}
	if event != 0x0 {
		fmt.Println("WaitForSingleObject returned unexpected value: %d", event)
	}
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
	si.AttributeList = make([]byte, lpSize, lpSize)
	ret, _, err := fInitializeProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&si.AttributeList[0])), 1, 0,
		uintptr(unsafe.Pointer(&lpSize)))
	if ret != 1 {
		return nil, fmt.Errorf("Failed to initialize thread attribute list: %v", err)
	}
	ret, _, err = fUpdateProcThreadAttribute.Call(uintptr(unsafe.Pointer(&si.AttributeList[0])),
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

func (c *ConPty) Read() (data []byte, err error) {
	if avail, err := c.dataAvailable(); avail <= 0 || err != nil {
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	maxRead := 1024 * 8
	numRead := uint32(0)
	buf := make([]byte, maxRead)
	err = windows.ReadFile(c.cmdOut, buf, &numRead, nil)
	if err != nil {
		return nil, err
	}
	return buf[:numRead], nil
}

func (c *ConPty) Write(data []byte) (written int32, err error) {
	numWritten := uint32(0)
	err = windows.WriteFile(c.cmdIn, data, &numWritten, nil)
	if err != nil {
		return -1, err
	}
	return int32(numWritten), nil
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
