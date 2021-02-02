package main

import (
	"os"
	"path/filepath"

	"k8s.io/klog/v2"

	"github.com/sjenning/kubechart/pkg/cmd"
	"github.com/sjenning/kubechart/pkg/cmd/kubechart"
)

func main() {
	defer klog.Flush()

	baseName := filepath.Base(os.Args[0])

	err := kubechart.NewCommand(baseName).Execute()
	cmd.CheckError(err)
}
