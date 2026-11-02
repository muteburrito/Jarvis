//go:build windows

package system

import (
	"os/exec"
	"syscall"
	"unsafe"
)

type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func configureHiddenCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
}

func detectRAMWindows() (totalMB, availMB int) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GlobalMemoryStatusEx")
	var mem memoryStatusEx
	mem.dwLength = uint32(unsafe.Sizeof(mem))
	ret, _, _ := proc.Call(uintptr(unsafe.Pointer(&mem)))
	if ret == 0 {
		return 0, 0
	}
	totalMB = int(mem.ullTotalPhys / (1024 * 1024))
	availMB = int(mem.ullAvailPhys / (1024 * 1024))
	return totalMB, availMB
}
