package event

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
)

type event struct {
	description string
	timestamp   time.Time
}

type store struct {
	sync.Mutex
	events map[string]map[string][]event
}

type Store interface {
	Add(namespace, name, value string)
	JSONHandler(w http.ResponseWriter, r *http.Request)
}

func NewStore() Store {
	return &store{events: map[string]map[string][]event{}}
}

func (s *store) Add(namespace, podname, description string) {
	s.Lock()
	defer s.Unlock()
	nsevents, ok := s.events[namespace]
	if !ok {
		s.events[namespace] = map[string][]event{}
		nsevents, _ = s.events[namespace]
	}
	podevents, ok := nsevents[podname]
	if !ok || podevents[len(podevents)-1].description != description {
		event := event{description, time.Now()}
		glog.Infof("adding event for %s/%s: %#v", namespace, podname, event)
		nsevents[podname] = append(nsevents[podname], event)
	} else {
		glog.Infof("duplicate event dropped for %s/%s", namespace, podname)
	}
}

type LabelData struct {
	TimeRange [2]time.Time `json:"timeRange,omitempty"`
	Val       string       `json:"val,omitempty"`
}

type GroupData struct {
	Label string      `json:"label,omitempty"`
	Data  []LabelData `json:"data,omitempty"`
}

type Group struct {
	Group string      `json:"group,omitempty"`
	Data  []GroupData `json:"data,omitempty"`
}

func (s *store) JSONHandler(w http.ResponseWriter, r *http.Request) {
	var groups []Group
	for group, labels := range s.events {
		g := Group{Group: group}
		for label, events := range labels {
			gd := GroupData{Label: label}
			for i, event := range events {
				ld := LabelData{TimeRange: [2]time.Time{event.timestamp, time.Now()}, Val: event.description}
				gd.Data = append(gd.Data, ld)
				if i > 0 {
					gd.Data[i-1].TimeRange[1] = event.timestamp
				}
			}
			g.Data = append(g.Data, gd)
		}
		groups = append(groups, g)
	}
	js, err := json.Marshal(groups)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
