package main

import (
	"os"

	"k8s.io/component-base/cli"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	"github.com/spiner-z/drift-k8s-scheduler/pkg/plugin"
)

func main() {
	cmd := app.NewSchedulerCommand(
		app.WithPlugin(plugin.PluginName, plugin.New),
	)
	os.Exit(cli.Run(cmd))
}
