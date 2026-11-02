//go:build !windows

package llamacpp

import "os/exec"

func configureChildProcess(cmd *exec.Cmd) {
}
