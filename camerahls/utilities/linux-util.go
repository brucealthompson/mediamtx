//go:build linux

package utilities

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/pipe.v2"
)

func AmAdmin() bool {
	uid := syscall.Geteuid()
	return uid == 0
}

func ExecHideWindow(installcmd *exec.Cmd) {
}

func RunMeElevated(args string) error {
	exe, _ := os.Executable()
	return RunElevated(exe, args)
}

func RunElevated(exe string, args string) error {
	var arglist []string = []string{"-S", exe}
	arglist = append(arglist, strings.Split(args, " ")...)
	done := make(chan error)
	a := fyne.CurrentApp()
	RootPassword(a, func(password string) {
		if len(password) > 0 {
			p := pipe.Line(
				pipe.Exec("echo", password),
				pipe.Exec("sudo", arglist...),
			)
			err := pipe.Run(p)
			done <- err
			return
		}
		done <- nil
	})
	return <-done
}

func RootPassword(a fyne.App, retemail func(string)) {
	window := a.NewWindow("Administrator Password")
	window.SetTitle("Administrator Password")
	formlabel := widget.NewLabel("Enter the administrator password for this computer")
	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("Password")
	loginForm := &widget.Form{
		SubmitText: "Accept",
		CancelText: "Cancel",
		OnCancel: func() {
			retemail("")
			window.Close()
		},
		OnSubmit: func() {
			retemail(password.Text)
			window.Close()
		},
	}
	logincontent := container.NewVBox(formlabel, loginForm)
	passworditem := widget.NewFormItem("Password", password)
	passworditem.HintText = "The root password for this computer"
	loginForm.AppendItem(passworditem)
	window.Resize(fyne.Size{Width: 300, Height: 200})
	window.SetCloseIntercept(func() {
		os.Exit(1)
	})
	window.SetContent(logincontent)
	window.Show()
}
