// +build windows

package service

import (
	"errors"
	"fmt"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Maximum amount of time to wait for service to reach desired state
const SERVICE_STATUS_TIMEOUT = 10

type Windows struct {
	Manager *mgr.Mgr
	logger  hclog.Logger
}

func ManagerConnect(logger hclog.Logger) (*Windows, error) {
	m, err := mgr.Connect()
	if err != nil {
		logger.Trace("windows service manager connect failure", "error", err)
		return nil, err
	}
	return &Windows{
		Manager: m,
		logger:  logger.Named("windows")}, nil
}

func (w *Windows) Disconnect() {
	w.Manager.Disconnect()
}

func (w *Windows) WaitForState(srv *mgr.Service, desiredState svc.State) error {
	waitInterval := 1 * time.Second
	for i := 0; i < SERVICE_STATUS_TIMEOUT; i++ {
		currentStatus, err := srv.Query()
		if err != nil {
			w.logger.Trace("service status query", "name", srv.Name, "error", err)
			return err
		}
		if currentStatus.State == desiredState {
			w.logger.Trace("service reached desired state", "name", srv.Name, "state", desiredState)
			return nil
		}
		w.logger.Trace("service not in desired state", "name", srv.Name, "current", currentStatus.State,
			"desired", desiredState)
		time.Sleep(waitInterval)
	}
	return errors.New("Service failed to reach desired state")
}

func (w *Windows) Get(name string) (*mgr.Service, error) {
	srv, err := w.Manager.OpenService(name)
	if err != nil {
		w.logger.Trace("service get failure", "name", name, "error", err)
		return nil, err
	}
	return srv, nil
}

func (w *Windows) IsRunning(name string) (bool, error) {
	srv, err := w.Get(name)
	if err != nil {
		return false, err
	}
	defer srv.Close()
	return w.IsRunningSvc(srv)
}

func (w *Windows) IsRunningSvc(s *mgr.Service) (bool, error) {
	status, err := s.Query()
	if err != nil {
		w.logger.Trace("service query failure", "name", s.Name, "error", err)
		return false, errors.New(fmt.Sprintf(
			"Failed to query %s: %s", s.Name, err))
	}
	if status.State == svc.Running {
		w.logger.Trace("service is running", "name", s.Name)
		return true, nil
	}
	w.logger.Trace("service is not running", "name", s.Name)
	return false, nil
}

func (w *Windows) IsStopped(name string) (bool, error) {
	srv, err := w.Get(name)
	if err != nil {
		return false, err
	}
	defer srv.Close()
	return w.IsStoppedSvc(srv)
}

func (w *Windows) IsStoppedSvc(s *mgr.Service) (bool, error) {
	status, err := s.Query()
	if err != nil {
		w.logger.Trace("service query failure", "name", s.Name, "error", err)
		return false, errors.New(fmt.Sprintf(
			"Failed to query %s: %s", s.Name, err))
	}
	if status.State == svc.Stopped {
		w.logger.Trace("service is stopped", "name", s.Name)
		return true, nil
	}
	w.logger.Trace("service is not stopped", "name", s.Name)
	return false, nil
}

func (w *Windows) Stop(name string) error {
	s, err := w.Get(name)
	if err != nil {
		return err
	}
	defer s.Close()
	return w.StopSvc(s)
}

func (w *Windows) StopSvc(s *mgr.Service) error {
	if stopped, _ := w.IsStoppedSvc(s); stopped {
		w.logger.Trace("service already stopped, not starting", "name", s.Name)
		return nil
	}
	if _, err := s.Control(svc.Stop); err != nil {
		w.logger.Trace("service control stop failure", "name", s.Name, "error", err)
		return err
	}
	if err := w.WaitForState(s, svc.Stopped); err != nil {
		w.logger.Trace("service stop wait failure", "name", s.Name, "error", err)
		return err
	}
	w.logger.Trace("service stopped", "name", s.Name)
	return nil
}

func (w *Windows) Start(name string) error {
	s, err := w.Get(name)
	if err != nil {
		return err
	}
	defer s.Close()
	return w.StartSvc(s)
}

func (w *Windows) StartSvc(s *mgr.Service) error {
	if running, _ := w.IsRunningSvc(s); running {
		w.logger.Trace("service already running, not starting", "name", s.Name)
		return nil
	}
	if err := s.Start(); err != nil {
		w.logger.Trace("service start failure", "name", s.Name, "error", err)
		return err
	}
	if err := w.WaitForState(s, svc.Running); err != nil {
		w.logger.Trace("service start wait failure", "name", s.Name, "error", err)
		return err
	}
	w.logger.Trace("service started", "name", s.Name)
	return nil
}
