package logger

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

type ObservedLog struct {
	Name    string
	Message string
	Level   slog.Level
	Time    time.Time
}

var observers = struct {
	sync.RWMutex
	values map[*observer]struct{}
}{values: make(map[*observer]struct{})}
var observerCount atomic.Int64

type observer struct{ emit func(ObservedLog) }

var createdObservers = struct {
	sync.RWMutex
	values map[*createdObserver]struct{}
}{values: make(map[*createdObserver]struct{})}
var createdObserverCount atomic.Int64

type createdObserver struct{ emit func(string) }

func ObserveCreated(emit func(string)) func() {
	if emit == nil {
		panic("nil logger observer")
	}
	entry := &createdObserver{emit: emit}
	createdObservers.Lock()
	createdObservers.values[entry] = struct{}{}
	createdObserverCount.Add(1)
	createdObservers.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			createdObservers.Lock()
			delete(createdObservers.values, entry)
			createdObserverCount.Add(-1)
			createdObservers.Unlock()
		})
	}
}

func observeCreated(name string) {
	if createdObserverCount.Load() == 0 {
		return
	}
	createdObservers.RLock()
	values := make([]*createdObserver, 0, len(createdObservers.values))
	for entry := range createdObservers.values {
		values = append(values, entry)
	}
	createdObservers.RUnlock()
	for _, entry := range values {
		entry.emit(name)
	}
}

func Observe(emit func(ObservedLog)) func() {
	if emit == nil {
		panic("nil log observer")
	}
	entry := &observer{emit: emit}
	observers.Lock()
	observers.values[entry] = struct{}{}
	observerCount.Add(1)
	observers.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() { observers.Lock(); delete(observers.values, entry); observerCount.Add(-1); observers.Unlock() })
	}
}

func observe(name, message string, record slog.Record) {
	if observerCount.Load() == 0 {
		return
	}
	observers.RLock()
	values := make([]*observer, 0, len(observers.values))
	for entry := range observers.values {
		values = append(values, entry)
	}
	observers.RUnlock()
	for _, entry := range values {
		entry.emit(ObservedLog{Name: name, Message: message, Level: record.Level, Time: record.Time})
	}
}
