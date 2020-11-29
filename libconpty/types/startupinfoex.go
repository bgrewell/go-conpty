package types

import "golang.org/x/sys/windows"

type StartupInfoEx struct {
	StartupInfo   windows.StartupInfo
	AttributeList []byte
}