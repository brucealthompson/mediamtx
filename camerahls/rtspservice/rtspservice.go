package rtspservice

import (
	"context"
	"errors"
	"log"
	"mediamtxhls/camerahls/utilities"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/kardianos/service"
	"github.com/shirou/gopsutil/process"
)

var (
	logger service.Logger
)

// Define Homecloud Remote Connector Service Start and Stop methods.
type program struct {
	exit chan struct{}
}

func (p *program) Start(s service.Service) error {
	if service.Interactive() {
		logger.Info("Running in terminal.")
	} else {
		logger.Info("Running under service manager.")
	}
	p.exit = make(chan struct{})
	// Start should not block. Do the actual work async.
	go func() {
		err := p.run()
		if err != nil {
			logger.Info(err.Error())
		}
	}()
	return nil
}

func GetRtspExe() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	exPath := filepath.Dir(ex)
	rtspcmd := filepath.Join(exPath, "mediamtx.exe")
	_, err = os.Stat(rtspcmd)
	if err != nil {
		return "", err
	}
	return rtspcmd, nil
}

func (p *program) run() error {
	logger.Info("Starting mediamtx service.")
	//time.Sleep(10 * time.Minute)
	rtspcmd, err := GetRtspExe()
	if err != nil {
		return err
	}
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	exPath := filepath.Dir(ex)
	outfile := filepath.Join(exPath, "mediamtx.out")
	os.Remove(outfile)
	cmd := exec.Command(rtspcmd)
	cmd.Dir = exPath
	utilities.ExecHideWindow(cmd)
	// open the out file for writing
	out, err := os.Create(outfile)
	if err != nil {
		return err
	}
	defer out.Close()
	cmd.Stdout = out
	cmd.Stderr = out
	err = cmd.Start()
	if err != nil {
		return err
	}
	cmd.Wait()
	return err
}

func processNameAlive(exename string) (bool, int) {
	processes, err := process.Processes()
	if err != nil {
		return false, -1
	}
	for _, p := range processes {
		processexe, err := p.Exe()
		if err != nil {
			continue
		}
		processpid := int(p.Pid)
		if processexe == exename && (os.Getpid() != processpid) {
			return true, processpid
		}
	}
	return false, -1
}

func (p *program) Stop(s service.Service) error {
	// Any work in Stop should be quick, usually a few seconds at most.
	logger.Info("mediamtx Service Stopping")
	rtspcmd, err := GetRtspExe()
	if err == nil {
		foundprocess, pid := processNameAlive(rtspcmd)
		if foundprocess {
			processdata, err := os.FindProcess(pid)
			if err == nil {
				processdata.Signal(os.Kill)
			}
		}
	}
	close(p.exit)
	return nil
}

func getServiceConfig(instance string) *service.Config {
	options := make(service.KeyValue)
	options["Restart"] = "on-success"
	options["SuccessExitStatus"] = "1 2 8 SIGKILL"
	display := "Camera HLS Service:" + instance
	serviceCfg := service.Config{
		Name:        "camerahls" + instance,
		DisplayName: display,
		Description: display,
		Dependencies: []string{
			"",
		},
		Option: options,
	}
	if runtime.GOOS == "linux" {
		serviceCfg.Dependencies = []string{
			"Requires=network.target",
			"After=network-online.target syslog.target"}
	}
	return &serviceCfg
}

func RtspServiceEnableStatus(instance string) (service.Status, error) {
	svcConfig := getServiceConfig(instance)
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return service.StatusUnknown, err
	}
	status, _ := s.Status()
	return status, nil
}

func ServiceControl(svcFlag *string, instance string) error {
	svcConfig := getServiceConfig(instance)
	prg := &program{}
	errs := make(chan error, 5)
	go func() {
		for {
			err := <-errs
			if err != nil {
				log.Print(err)
			}
		}
	}()
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Print(err)
		return err
	}
	logger, err = s.Logger(errs)
	if err != nil {
		log.Print(err)
		return err
	}
	if len(*svcFlag) != 0 {
		// Channel used to receive the result from background function
		ch := make(chan string, 1)
		// Create a context with a timeout
		ctxTimeout, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		defer cancel()
		go func() {
			err := service.Control(s, *svcFlag)
			if err != nil {
				log.Print(err)
				ch <- err.Error()
			} else {
				ch <- ""
			}
		}()
		select {
		case <-ctxTimeout.Done():
			return errors.New("Timeout waiting for connector service")
		case errstring := <-ch:
			if len(errstring) > 0 {
				return errors.New(errstring)
			} else {
				return nil
			}
		}
	}
	err = s.Run()
	if err != nil {
		log.Print(err)
	}
	return (err)
}
