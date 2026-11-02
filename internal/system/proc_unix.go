//go:build !windows

package system

import "os/exec"

func configureHiddenCommand(cmd *exec.Cmd) {
}

func detectRAMWindows() (totalMB, availMB int) {
	return 0, 0
}
