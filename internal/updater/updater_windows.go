package updater

import (
	"os/exec"
	"syscall"
)

func launchInstaller(path string) error {
	cmd := exec.Command(path, "/S")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
