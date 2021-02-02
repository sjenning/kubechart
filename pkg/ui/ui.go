package ui

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/sjenning/kubechart/pkg/event"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
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
		cachedLog, ok := store.GetLog(vars["namespace"], vars["podname"])
		if ok {
			io.WriteString(w, cachedLog)
		}
	})
	klog.Infof(fmt.Sprintf("Listening on :%d", port))
	klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
