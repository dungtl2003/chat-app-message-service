package services

type ServiceStatus int

const (
	READY ServiceStatus = iota
	ERROR
	STOPPED
)

type Service interface {
	Status() ServiceStatus
	Name() string
	Close() error
}
