# kubechart
Bootchart for Kubernetes Pods

![KubeChart](https://raw.githubusercontent.com/sjenning/kubechart/master/kubechart.png)

## Building

`go build ./cmd/kubechart`

## Running

Set your `KUBECONFIG` appropriately or using the `--kubeconfig` flag to pass it directly

`./kubechart`

Then open up http://localhost:3000 to see the KubeChart.

If it doesn't work, add `--logtostderr` to the flag to get more verbose logging.
