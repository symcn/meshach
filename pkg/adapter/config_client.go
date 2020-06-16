package adapter

import "github.com/mesh-operator/pkg/adapter/events"

type ConfigurationCenterClient interface {
	Start() error

	Events() <-chan *events.ConfigEvent

	Stop() error
}
