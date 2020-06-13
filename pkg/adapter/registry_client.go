package adapter

type RegistryClient interface {
	Start() error

	Events() <-chan ServiceEvent

	Stop()
}
