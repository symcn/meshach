package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/symcn/mesh-operator/cmd/mesh-operator/app"
	_ "github.com/symcn/mesh-operator/pkg/adapter/configcenter/zk"
	_ "github.com/symcn/mesh-operator/pkg/adapter/registry/zk"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	rootCmd := app.GetRootCmd(os.Args[1:])

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(-1)
	}
}
