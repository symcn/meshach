package adapter

type ConfigurationCenterClient interface {
	Start() error

	Events() <-chan interface{}

	Stop() error
}
