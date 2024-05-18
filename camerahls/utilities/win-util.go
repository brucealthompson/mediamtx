//go:build windows

package utilities

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func ExecHideWindow(installcmd *exec.Cmd) {
	installcmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func AmAdmin() bool {
	elevated := windows.GetCurrentProcessToken().IsElevated()
	return elevated
}

func RunMeElevated(args string) error {
	exe, _ := os.Executable()
	return RunElevated(exe, args)
}

func RunElevated(exe string, args string) error {
	verb := "runas"
	cwd, _ := os.Getwd()

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exe)
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	var showCmd int32 = 1 //SW_NORMAL

	err := windows.ShellExecute(0, verbPtr, exePtr, argPtr, cwdPtr, showCmd)
	return err
}
