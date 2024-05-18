//go:build android

package utilities

import (
	"errors"
	"os/exec"
)

func ExecHideWindow(installcmd *exec.Cmd) {
}

func AmAdmin() bool {
	return true
}

func RunMeElevated(args string) error {
	return errors.New("Not supported on this operating system")
}
