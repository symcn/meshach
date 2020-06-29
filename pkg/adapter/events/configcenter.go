package events

type ConfigurationCenter interface {
	Start() error

	Events() <-chan *ConfigEvent

	FindConfiguratorConfig(serviceName string) *ConfiguratorConfig

	Stop() error
}
