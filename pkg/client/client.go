package client

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/sjenning/kubechart/pkg/version"
)

// Config returns a *rest.Config, using either the kubeconfig (if specified) or an in-cluster configuration.
func Config(kubeconfig, baseName string) (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = kubeconfig
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	clientConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	clientConfig.UserAgent = buildUserAgent(
		baseName,
		version.Version,
		version.FormattedGitSHA(),
		runtime.GOOS,
		runtime.GOARCH,
	)

	return clientConfig, nil
}

// buildUserAgent builds a User-Agent string from given args.
func buildUserAgent(command, version, formattedSha, os, arch string) string {
	return fmt.Sprintf(
		"%s/%s (%s/%s) %s", command, version, os, arch, formattedSha)
}
