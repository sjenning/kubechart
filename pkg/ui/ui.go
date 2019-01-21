package ui

import (
	"io"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/sjenning/kubechart/pkg/event"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func Run(store event.Store, client kubernetes.Interface) {
	r := mux.NewRouter()
	r.Handle("/", http.FileServer(http.Dir("./static")))
	r.HandleFunc("/data.json", store.JSONHandler)
	r.HandleFunc("/logs/{namespace}/{podname}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		namespace, podname := vars["namespace"], vars["podname"]
		req := client.CoreV1().Pods(namespace).GetLogs(podname, &v1.PodLogOptions{})
		podLogs, err := req.Stream()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		defer podLogs.Close()
		_, err = io.Copy(w, podLogs)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	})
	glog.Infof("Listening on :3000")
	http.ListenAndServe(":3000", r)
}
