// Homecloud remote connector service
package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"mediamtxhls/camerahls/rtspservice"
	"mediamtxhls/camerahls/utilities"

	"github.com/kardianos/service"
)

const (
	ServiceFlag = "service"
	BeginCmd    = "begin"
	EndCmd      = "end"
)

func serviceInstall(instance string) {
	flag := "install"
	rtspservice.ServiceControl(&flag, instance)
}

func serviceStart(instance string) {
	flag := "start"
	rtspservice.ServiceControl(&flag, instance)
}

func serviceStop(instance string) {
	flag := "stop"
	rtspservice.ServiceControl(&flag, instance)
}

func serviceUninstall(instance string) {
	flag := "uninstall"
	rtspservice.ServiceControl(&flag, instance)
}

func getMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash[0:4])
}

// Service setup.
//
//	Define service config.
//	Create the service.
//	Setup the logger.
//	Handle service controls (optional).
//	Run the service.
func main() {
	svcFlag := flag.String(ServiceFlag, "", "Control the camera service.")
	flag.Parse()
	ex, err := os.Executable()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	instance := getMD5Hash(ex)
	if !utilities.AmAdmin() {
		fmt.Println("Camera HLS service must run at elevated priviledge")
		os.Exit(1)
	}
	status, err := rtspservice.RtspServiceEnableStatus(instance)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	switch *svcFlag {
	case BeginCmd:
		switch status {
		case service.StatusStopped:
			serviceUninstall(instance)
			serviceInstall(instance)
			serviceStart(instance)
		case service.StatusRunning:
			serviceStop(instance)
			serviceUninstall(instance)
			serviceInstall(instance)
			serviceStart(instance)
		case service.StatusUnknown:
			serviceInstall(instance)
			serviceStart(instance)
		default:
			fmt.Println("Invalid Service status")
			os.Exit(1)
		}
	case EndCmd:
		switch status {
		case service.StatusRunning:
			serviceStop(instance)
			serviceUninstall(instance)
		case service.StatusStopped:
			serviceUninstall(instance)
		case service.StatusUnknown:
		default:
			fmt.Println("Invalid Service status")
			os.Exit(1)
		}
	case "":
		rtspservice.ServiceControl(svcFlag, instance)
	case "install", "uninstall", "start", "stop", "pause", "continue":
		rtspservice.ServiceControl(svcFlag, instance)
	default:
		fmt.Println("Invalid Service command - " + *svcFlag)
		os.Exit(1)
	}
}
