package utilities

import (
	"bytes"
	"errors"
	"os/exec"
	"runtime"
	"strconv"
)

const firewallcmd = "netsh.exe"

func OpenFirewallPort(rulename string, serverport int) error {
	switch runtime.GOOS {
	case "windows":
		//netsh advfirewall firewall add rule name= dir=in action=allow protocol=TCP localport=
		portstring := strconv.Itoa(serverport)
		params := []string{"advfirewall", "firewall", "add", "rule", "name=" + rulename, "dir=in", "action=allow", "protocol=TCP", "localport=" + portstring}
		if AmAdmin() {
			var stderr bytes.Buffer
			cmd := exec.Command(firewallcmd, params...)
			ExecHideWindow(cmd)
			cmd.Stderr = &stderr
			err := cmd.Run()
			errstring := stderr.String()
			if err != nil {
				return err
			} else if len(errstring) > 0 {
				return errors.New(errstring)
			}
		}
	default:
		return errors.New("add firewall rule not supported")
	}
	return nil
}

func CloseFirewallPort(rulename string) error {
	if runtime.GOOS == "windows" {
		//netsh advfirewall firewall delete rule name=""
		params := []string{"advfirewall", "firewall", "rule", "rule", "name=" + rulename}
		if AmAdmin() {
			var stderr bytes.Buffer
			cmd := exec.Command(firewallcmd, params...)
			ExecHideWindow(cmd)
			cmd.Stderr = &stderr
			err := cmd.Run()
			errstring := stderr.String()
			if err != nil {
				return err
			} else if len(errstring) > 0 {
				return errors.New(errstring)
			}
		}
	}
	return errors.New("remove firewall rule not supported")
}
