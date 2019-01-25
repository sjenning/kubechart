package event

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/sjenning/kubechart/pkg/log"

	"k8s.io/client-go/kubernetes"
)

type event struct {
	description string
	timestamp   time.Time
}

type cacheEntry struct {
	namespace string
	podname string
	bytes int
}

type store struct {
	sync.Mutex
	events map[string]map[string][]event
	// TBD: this will consume lots o' memory
	logCacheLimit int
	client kubernetes.Interface
	logCache map[string]map[string]string
	logCacheSize int
	logCacheEntries []cacheEntry
}

type Store interface {
	Add(namespace, podname, value string)
	GetCachedLog(namespace, podname string) (string, bool)
	JSONHandler(w http.ResponseWriter, r *http.Request)
	addCacheEntry(namespace, podname, data string)
	deleteCacheEntry(namespace, podname string)
}

func NewStore(client kubernetes.Interface, cacheLimit int) Store {
	return &store{
		events: map[string]map[string][]event{},
		logCacheLimit: cacheLimit,
		client: client,
		logCache: map[string]map[string]string{},
	}
}

// Store must be locked!
func (s *store) deleteCacheEntry(namespace, podname string) {
	nscache, ok := s.logCache[namespace]
	if !ok {
		return
	}
	_, ok = nscache[podname]
	if !ok {
		return
	}
	delete(nscache, podname)
	cacheEntries := len(s.logCacheEntries)
	for idx, cacheEntry := range s.logCacheEntries {
		if cacheEntry.namespace == namespace && cacheEntry.podname == podname {
			s.logCacheSize -= cacheEntry.bytes
			if cacheEntries == 1 {
				s.logCacheEntries = nil
			} else if idx == 0 {
				s.logCacheEntries = s.logCacheEntries[1:]
			} else if idx == cacheEntries - 1 {
				s.logCacheEntries = s.logCacheEntries[0:cacheEntries - 2]
			} else {
				s.logCacheEntries = append(s.logCacheEntries[0:idx - 1], s.logCacheEntries[idx + 1:cacheEntries - 1]...)
			}
		}
	}
}

func (s *store) addCacheEntry(namespace, podname, data string) {
	nscache, ok := s.logCache[namespace]
	if !ok {
		s.logCache[namespace] = map[string]string{}
		nscache, _ = s.logCache[namespace]
	}
	odata, _ := nscache[podname]
	nscache[podname] = odata + data
	s.logCacheEntries = append(s.logCacheEntries, cacheEntry{
		namespace: namespace,
		podname: podname,
		bytes: len(nscache[podname]),
	})
	s.logCacheSize += len(data)
	for s.logCacheSize > s.logCacheLimit {
		head := s.logCacheEntries[0]
		s.deleteCacheEntry(head.namespace, head.podname)
	}
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
		if ok {
			lastEvent := podevents[len(podevents)-1]
			if s.logCacheLimit > 0 && lastEvent.description == "Running" {
				logString, err := log.LogPodToString(s.client, namespace, podname)
				if len(logString) > 0 && err == nil {
					s.addCacheEntry(namespace, podname, logString)
				}
			}
		}
		nsevents[podname] = append(nsevents[podname], event)
	} else {
		glog.Infof("duplicate event dropped for %s/%s\n", namespace, podname)
	}
}

func (s *store) GetCachedLog(namespace, podname string) (string, bool) {
	s.Lock()
	defer s.Unlock()
	nscache, ok := s.logCache[namespace]
	if !ok {
		return "", false
	}
	cache, ok := nscache[podname]
	return cache, ok
}

type LabelData struct {
	TimeRange [2]time.Time `json:"timeRange,omitempty"`
	Val       string       `json:"val"`
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
