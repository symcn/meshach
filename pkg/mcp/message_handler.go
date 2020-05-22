package mcp

import (
	"google.golang.org/grpc"
	mcp "istio.io/api/mcp/v1alpha1"
)

type Handler struct {
	grpc.ClientStream
}

func (h *Handler) Send(*mcp.RequestResources) error {
	return nil
}
func (h *Handler) Recv() (*mcp.Resources, error) {
	return &mcp.Resources{}, nil
}
