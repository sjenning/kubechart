package ui

import (
	"net/http"

	"github.com/golang/glog"
	"github.com/sjenning/kubechart/pkg/event"
)

func Run(store event.Store) {
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/data", store.JSONHandler)
	glog.Infof("Listening on :3000")
	http.ListenAndServe(":3000", nil)
}
