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
	"k8s.io/client-go/kubernetes"
)

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
		req := client.CoreV1().Pods(namespace).GetLogs(podname, &v1.PodLogOptions{})
		podLogs, err := req.Stream()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer podLogs.Close()
		_, err = io.Copy(w, podLogs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	glog.Infof(fmt.Sprintf("Listening on :%d", port))
	http.ListenAndServe(fmt.Sprintf(":%d", port), r)
}
