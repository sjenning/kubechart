package ui

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/sjenning/kubechart/pkg/event"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func printOneLog(client kubernetes.Interface, w http.ResponseWriter, name, namespace, container string, idx, total int) error {
	pod := client.CoreV1().Pods(namespace)
	if total > 1 {
		io.WriteString(w, "================================================================\n")
		io.WriteString(w, fmt.Sprintf("Container %d/%d: %s\n", idx+1, total, container))
		io.WriteString(w, "================================================================\n\n")
	}
	req := pod.GetLogs(name, &v1.PodLogOptions{Container: container})
	podLogs, err := req.Stream()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	defer podLogs.Close()
	_, err = io.Copy(w, podLogs)
	if total > 1 && idx < total-1 {
		io.WriteString(w, "\n\n")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	return nil
}

func Run(store event.Store, client kubernetes.Interface, port uint16) {
	r := mux.NewRouter()
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		path = "."
	}
	r.Handle("/", http.FileServer(http.Dir(fmt.Sprintf("%s/static", path))))
	r.HandleFunc("/data.json", store.JSONHandler)
	r.HandleFunc("/logs/{namespace}/{podname}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		namespace, podname := vars["namespace"], vars["podname"]
		pod := client.CoreV1().Pods(namespace)
		pods, _ := pod.List(metav1.ListOptions{})
		for _, item := range pods.Items {
			if item.Name == podname {
				cCount := len(item.Spec.Containers)
				for idx, container := range item.Spec.Containers {
					_ = printOneLog(client, w, podname, namespace, container.Name, idx, cCount)
					if err != nil {
						io.WriteString(w, fmt.Sprintf("Error: %#+v", err))
						break
					}
				}
				break
			}
		}
	})
	glog.Infof(fmt.Sprintf("Listening on :%d", port))
	glog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
