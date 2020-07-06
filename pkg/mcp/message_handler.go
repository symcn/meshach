package mcp

import (
	"google.golang.org/grpc"
	mcp "istio.io/api/mcp/v1alpha1"
)

// Handler ...
type Handler struct {
	grpc.ClientStream
}

// Send ...
func (h *Handler) Send(*mcp.RequestResources) error {
	return nil
}

// Recv ...
func (h *Handler) Recv() (*mcp.Resources, error) {
	return &mcp.Resources{}, nil
}
