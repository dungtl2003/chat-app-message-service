package services

import "fmt"

type ServiceStatus int

const (
	ServiceReady ServiceStatus = iota
	ServiceError
	ServiceStopped
)

type ServiceNotRunningError struct {
	ServiceName string
}

func (e ServiceNotRunningError) Error() string {
	return fmt.Sprintf("service %s is not running", e.ServiceName)
}

type ServiceNotReadyError struct {
	ServiceName string
}

func (e ServiceNotReadyError) Error() string {
	return fmt.Sprintf("service %s is not ready", e.ServiceName)
}

type Service interface {
	Status() ServiceStatus
	Name() string
	Close() error
}
