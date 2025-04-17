package services

type ServiceStatus int

const (
	RUNNING ServiceStatus = iota
	INITIALIZING
	READY
	ERROR
	STOPPED
)

type Service interface {
	GetStatus() ServiceStatus
	GetName() string
	Close() error
	Run() error
}
