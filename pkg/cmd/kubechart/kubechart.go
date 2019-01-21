package kubechart

import (
	"flag"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/informers"

	"github.com/sjenning/kubechart/pkg/client"
	"github.com/sjenning/kubechart/pkg/cmd"
	"github.com/sjenning/kubechart/pkg/controller"
	"github.com/sjenning/kubechart/pkg/event"
	"github.com/sjenning/kubechart/pkg/signals"
	"github.com/sjenning/kubechart/pkg/ui"
)

func NewCommand(name string) *cobra.Command {
	f := client.NewFactory(name)

	c := &cobra.Command{
		Use:   name,
		Short: "Monitor Pod phase transitions over time in a Kubernetes cluster.",
		Run: func(c *cobra.Command, args []string) {
			cmd.CheckError(run(c, f))
		},
	}

	f.BindFlags(c.PersistentFlags())
	c.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	return c
}

func run(c *cobra.Command, f client.Factory) error {
	stopCh := signals.SetupSignalHandler()
	client, err := f.Client()
	if err != nil {
		return err
	}
	informerFactory := informers.NewSharedInformerFactory(client, time.Second*30)
	podInformer := informerFactory.Core().V1().Pods()
	eventStore := event.NewStore()
	controller := controller.New(client, podInformer, eventStore)
	if err != nil {
		return err
	}
	informerFactory.Start(stopCh)
	go ui.Run(eventStore, client)
	if err = controller.Run(4, stopCh); err != nil {
		return err
	}
	return nil
}
