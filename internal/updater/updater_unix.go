//go:build !windows

package updater

import "fmt"

func launchInstaller(path string) error {
	return fmt.Errorf("automatic installer launch is not supported on this platform: %s", path)
}
