package mcp

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func Test_ConnectNacosAsMcpServer(t *testing.T) {
	c, err := NewClient()
	ctx := context.Background()

	go func() {
		c.Run(ctx)
	}()

	time.Sleep(10 * time.Minute)
	ctx.Done()

	assert.Nil(t, err)
}
