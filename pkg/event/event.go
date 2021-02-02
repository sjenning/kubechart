package event

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sjenning/kubechart/pkg/log"
	"k8s.io/klog/v2"

	"k8s.io/client-go/kubernetes"
)

type event struct {
	description string
	timestamp   time.Time
}

type cacheType int

const (
	logEntry   cacheType = iota
	stateEntry cacheType = iota
)

type cacheInsertMode int

const (
	cacheNewEntry   cacheInsertMode = iota
	cacheAppendLog  cacheInsertMode = iota
	cacheReplaceLog cacheInsertMode = iota
)

type cacheEntry struct {
	cachetype cacheType
	logdata   string
}

type store struct {
	sync.Mutex
	events       map[string]map[string][]event
	client       kubernetes.Interface
	logCache     map[string]map[string][]cacheEntry
	lastLogEntry map[string]map[string]int
	logAllEvents bool
}

type Store interface {
	Add(namespace, podname, value, message string)
	GetLog(namespace, podname string) (string, bool)
	JSONHandler(w http.ResponseWriter, r *http.Request)
	addCacheEntry(namespace, podname, data string, logtype cacheType, insertMode cacheInsertMode)
}

func NewStore(client kubernetes.Interface, logAllEvents bool) Store {
	return &store{
		events:       map[string]map[string][]event{},
		client:       client,
		logCache:     map[string]map[string][]cacheEntry{},
		lastLogEntry: map[string]map[string]int{},
		logAllEvents: logAllEvents,
	}
}

// Store must be locked!

func (s *store) getCurrentState(namespace, podname string) string {
	nsevents, ok := s.events[namespace]
	if !ok {
		return ""
	}
	podevents, ok := nsevents[podname]
	if !ok {
		return ""
	}
	return podevents[len(podevents)-1].description
}

func (s *store) addCacheEntry(namespace, podname, data string, logtype cacheType, insertMode cacheInsertMode) {
	if len(data) == 0 {
		return
	}
	nscache, ok := s.logCache[namespace]
	if !ok {
		s.logCache[namespace] = map[string][]cacheEntry{}
		s.lastLogEntry[namespace] = map[string]int{}
		nscache, _ = s.logCache[namespace]
	}
	_, ok = nscache[podname]
	if !ok {
		nscache[podname] = []cacheEntry{}
		s.lastLogEntry[namespace][podname] = -1
	}
	if logtype == logEntry {
		if insertMode == cacheReplaceLog && s.lastLogEntry[namespace][podname] >= 0 {
			odata := nscache[podname][s.lastLogEntry[namespace][podname]].logdata
			// Don't replace data with something shorter
			if len(odata) > len(data) {
				insertMode = cacheNewEntry
			} else {
				// If the beginnings of the two entries don't match,
				// create a new entry so we don't lose anything
				bytesToCompare := 1024
				if len(odata) < bytesToCompare {
					bytesToCompare = len(odata)
				}
				if odata[:bytesToCompare] != data[:bytesToCompare] {
					insertMode = cacheNewEntry
				}
			}
		}
		if insertMode == cacheNewEntry || s.lastLogEntry[namespace][podname] == -1 {
			nscache[podname] = append(nscache[podname], cacheEntry{cachetype: logEntry, logdata: data})
			s.lastLogEntry[namespace][podname] = len(nscache[podname]) - 1
		} else {
			if insertMode == cacheReplaceLog {
				nscache[podname][s.lastLogEntry[namespace][podname]].logdata = data
			} else {
				nscache[podname][s.lastLogEntry[namespace][podname]].logdata = nscache[podname][s.lastLogEntry[namespace][podname]].logdata + data
			}
		}
	} else {
		nscache[podname] = append(nscache[podname], cacheEntry{cachetype: stateEntry, logdata: data})
		s.lastLogEntry[namespace][podname] = len(nscache[podname]) - 1
	}
}

func (s *store) Add(namespace, podname, description, message string) {
	s.Lock()
	defer s.Unlock()
	nsevents, ok := s.events[namespace]
	if !ok {
		s.events[namespace] = map[string][]event{}
		nsevents, _ = s.events[namespace]
	}
	podevents, ok := nsevents[podname]
	lastDescription := "(created)"
	if ok {
		lastDescription = podevents[len(podevents)-1].description
	}
	event := event{description, time.Now()}
	klog.Infof("adding event for %s/%s: %#v", namespace, podname, event)
	if lastDescription == "Running" {
		logString, err := log.LogPodToString(s.client, namespace, podname)
		if err == nil && len(logString) > 0 {
			if event.description == "Running" {
				s.addCacheEntry(namespace, podname, logString, logEntry, cacheReplaceLog)
			} else {
				s.addCacheEntry(namespace, podname, logString, logEntry, cacheNewEntry)
			}
		}
	}
	if lastDescription != description || s.logAllEvents {
		entry := fmt.Sprintf(">>> %v %s -> %s\n", event.timestamp, lastDescription, event.description)
		if message != "" {
			entry = fmt.Sprintf("%s\nMessage:\n%s\n\n", entry, message)
		}
		s.addCacheEntry(namespace, podname, entry, stateEntry, cacheNewEntry)
		nsevents[podname] = append(nsevents[podname], event)
	}
	if lastDescription != "Running" {
		s.lastLogEntry[namespace][podname] = -1
	}
}

func (s *store) GetLog(namespace, podname string) (string, bool) {
	s.Lock()
	state := s.getCurrentState(namespace, podname)
	s.Unlock()
	data := ""
	// Drop the lock while we retrieve data; this may be expensive.
	// If the state changes at the wrong time, we may get duplicate log
	// data or miss some, which is not a serious problem.
	if state == "Running" {
		// Don't worry about errors here; we just won't return any new date.
		data, _ = log.LogPodToString(s.client, namespace, podname)
	}
	s.Lock()
	defer s.Unlock()
	if data != "" {
		s.addCacheEntry(namespace, podname, data, logEntry, cacheReplaceLog)
	}
	nscache, ok := s.logCache[namespace]
	if !ok {
		return "", false
	}
	cache, ok := nscache[podname]
	var b strings.Builder
	for _, entry := range cache {
		b.WriteString(entry.logdata)
	}
	return b.String(), ok
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
