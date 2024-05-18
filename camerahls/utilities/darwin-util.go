//go:build darwin

package utilities

import (
	"encoding/hex"
	"errors"
	"net"
	"os/exec"
	"strings"
	"syscall"

	"github.com/denisbrodbeck/machineid"
)

func GetHomecldExe() string {
	return "Homecloud Monitor"
}

func GetVlcPath() (string, error) {
	return "", errors.New("VLC is not installed")
}

func GetHomecldUrl() (string, error) {
	return "https://storage.googleapis.com/homecld/Homecloud%20Monitor.dmg", nil
}

func ExecHideWindow(installcmd *exec.Cmd) {
}

func AmAdmin() bool {
	uid := syscall.Geteuid()
	return uid == 0
}

func LocalSubnet(localIP net.IP) net.IPNet {
	var localAddr net.IPNet
	localAddr.IP = localIP
	out, err := exec.Command("ifconfig").CombinedOutput()
	if err != nil {
		localAddr.Mask = localIP.DefaultMask()
		return localAddr
	}
	lines := strings.Split(string(out[:]), "\n")
	ipstring := localIP.String()
	for _, line := range lines {
		words := strings.Fields(line)
		if len(words) < 4 {
			continue
		}
		if (words[0] == "inet") && (words[1] == ipstring) && (words[2] == "netmask") {
			localAddr.Mask, err = hex.DecodeString(strings.TrimLeft(words[3], "0x"))
			if err == nil {
				return localAddr
			}
		}
	}
	localAddr.Mask = localIP.DefaultMask()
	return localAddr
}

func GetMachineUuid() (string, error) {
	uuid, err := machineid.ID()
	if err != nil {
		return "", err
	}
	return uuid, nil
}

func MakeLink(src, dst string) error {
	return (errors.New("Not implemented"))
}
