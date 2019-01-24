package client

import (
	"fmt"

	"k8s.io/client-go/kubernetes"

	"github.com/spf13/pflag"
)

const (
	defaultHttpPort = 3000
)

// Factory knows how to create a Kubernetes client.
type Factory interface {
	// BindFlags binds common flags (--kubeconfig, --namespace) to the passed-in FlagSet.
	BindFlags(flags *pflag.FlagSet)
	// KubeClient returns a Kubernetes client. It uses the following priority to specify the cluster
	// configuration: --kubeconfig flag, KUBECONFIG environment variable, in-cluster configuration.
	Client() (kubernetes.Interface, error)
	// Port returns the port to listen on
	Port() uint16
}

type factory struct {
	flags      *pflag.FlagSet
	kubeconfig string
	baseName   string
	httpPort   uint16
}

// NewFactory returns a Factory.
func NewFactory(baseName string) Factory {
	f := &factory{
		flags:    pflag.NewFlagSet("", pflag.ContinueOnError),
		baseName: baseName,
	}

	f.flags.StringVar(&f.kubeconfig, "kubeconfig", "", "Path to the kubeconfig file to use to talk to the Kubernetes apiserver. If unset, try the environment variable KUBECONFIG, as well as in-cluster configuration")
	f.flags.Uint16Var(&f.httpPort, "http-port", uint16(defaultHttpPort), fmt.Sprintf("Port to serve charts on.", defaultHttpPort))

	return f
}

func (f *factory) BindFlags(flags *pflag.FlagSet) {
	flags.AddFlagSet(f.flags)
}

func (f *factory) Client() (kubernetes.Interface, error) {
	clientConfig, err := Config(f.kubeconfig, f.baseName)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (f *factory) Port() uint16 {
	return f.httpPort
}
