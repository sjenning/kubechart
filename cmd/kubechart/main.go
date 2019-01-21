package main

import (
	"os"
	"path/filepath"

	"github.com/golang/glog"

	"github.com/sjenning/kubechart/pkg/cmd"
	"github.com/sjenning/kubechart/pkg/cmd/kubechart"
)

func main() {
	defer glog.Flush()

	baseName := filepath.Base(os.Args[0])

	err := kubechart.NewCommand(baseName).Execute()
	cmd.CheckError(err)
}
