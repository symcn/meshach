package mcp

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	mcp "istio.io/api/mcp/v1alpha1"
	"istio.io/istio/pilot/pkg/features"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/istio/pkg/mcp/sink"
	"istio.io/istio/pkg/mcp/testing/monitoring"
)

type clientStreamCreator struct {
}

// EstablishResourceStream ...
func (c *clientStreamCreator) EstablishResourceStream(ctx context.Context, opts ...grpc.CallOption) (mcp.ResourceSource_EstablishResourceStreamClient, error) {
	handler := &Handler{}
	return handler, nil
}

// NewClient ...
func NewClient() (client *sink.Client, err error) {
	_, cancel := context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	// var serverAddr = "10.248.224.144:8080"
	var serverAddr = "10.12.210.41:20002"
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("Error connecting to server:%v", err)
	}

	//colOptions := []sink.CollectionOptions{}
	//colOptions = append(colOptions, sink.CollectionOptions{
	//	Name:        "istio/networking/v1alpha3/serviceentries",
	//	Incremental: false,
	//})
	//colOptions = append(colOptions, sink.CollectionOptions{
	//	Name:        "istio/networking/v1alpha3/virtualservices",
	//	Incremental: false,
	//})
	all := collections.Pilot.All()
	colOptions := make([]sink.CollectionOptions, 0, len(all))
	for _, c := range all {
		colOptions = append(colOptions, sink.CollectionOptions{Name: c.Name().String(), Incremental: features.EnableIncrementalMCP})
	}

	u := sink.NewInMemoryUpdater()

	options := &sink.Options{
		CollectionOptions: colOptions,
		Updater:           u,
		ID:                "foo",
		// Metadata:          nil,
		Reporter: monitoring.NewInMemoryStatsContext(),
	}

	cl := mcp.NewResourceSourceClient(conn)
	//cl := &clientStreamCreator{}

	// rsClient, err :=
	// cl.EstablishResourceStream(context.Background())
	// rsClient.SendMsg()

	c := sink.NewClient(cl, options)

	return c, nil

}
