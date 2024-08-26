package properties

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/pflag"
)

var commandlineProperties map[string]string

var Global = &Properties{
	m: make(map[string]string),
}

var LoadFile = func(filename string) error {
	return Global.LoadFile(filename)
}

func BindFlags(flags *pflag.FlagSet) {
	flags.StringToStringVarP(&commandlineProperties, "properties", "P", nil, "System properties")
}

type Properties struct {
	m         map[string]string
	filename  string
	listeners []func(*Properties)
	lock      sync.RWMutex
	close     func()
	Reload    func()
}

func (p *Properties) RegisterListener(fn func(*Properties)) {
	p.listeners = append(p.listeners, fn)
}

func (p *Properties) Set(key string, value any) {
	p.lock.Lock()
	defer p.notify()
	defer p.lock.Unlock()
	p.m[key] = fmt.Sprintf("%v", value)
}

func (p *Properties) notify() {
	for _, listener := range p.listeners {
		listener(p)
	}
}

func (p *Properties) GetAll() map[string]string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	m := p.m
	for k, v := range commandlineProperties {
		m[k] = v
	}
	return m
}

func (p *Properties) Get(key string) string {
	if v, ok := commandlineProperties[key]; ok {
		return v
	}

	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.m[key]
}

func (p *Properties) Update(props map[string]string) {
	defer p.notify()

	p.lock.Lock()
	defer p.lock.Unlock()

	for k, v := range props {
		p.m[k] = v
	}
}

func (p *Properties) LoadFile(filename string) error {
	if !path.IsAbs(filename) {
		cwd, _ := os.Getwd()
		filename = path.Join(cwd, filename)
	}
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		slog.Warn(fmt.Sprintf("%s does not exist", filename))
		p.Update(commandlineProperties)
		return nil
	} else if err != nil {
		return err
	}
	defer file.Close()
	p.filename = filename

	if p.close == nil {
		p.close = p.Watch()
	}

	slog.Info(fmt.Sprintf("Loading properties from %s", filename))
	var props = make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		tokens := strings.SplitN(line, "=", 2)
		if len(tokens) != 2 {
			return fmt.Errorf("invalid line: %s", line)
		}

		key := strings.TrimSpace(tokens[0])
		value := strings.TrimSpace(tokens[1])
		props[key] = value
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}

	for k, v := range commandlineProperties {
		props[k] = v
	}
	p.Update(props)

	return nil
}

func RegisterListener(fn func(*Properties)) {
	Global.RegisterListener(fn)
}

func Set(key string, value any) {
	Global.Set(key, value)
}

func Get(key string) string {
	return Global.Get(key)
}

func Update(props map[string]string) {
	Global.Update(props)
}

func On(def bool, keys ...string) bool {
	return Global.On(def, keys...)
}
func Duration(def time.Duration, keys ...string) time.Duration {
	return Global.Duration(def, keys...)
}

func String(def string, keys ...string) string {
	return Global.String(def, keys...)
}

func Int(def int, key string) int {
	return Global.Int(def, key)
}

func (p *Properties) On(def bool, keys ...string) bool {
	for _, key := range keys {
		if v := p.Get(key); v != "" {
			return strings.ToLower(v) == "true"
		}
	}
	return def
}

func (p *Properties) String(def string, keys ...string) string {
	for _, key := range keys {
		if v := p.Get(key); v != "" {
			return v
		}
	}
	return def
}

func (p *Properties) Duration(def time.Duration, keys ...string) time.Duration {
	for _, key := range keys {
		if v := p.Get(key); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				return d
			}
			//FIXME: return the failed parsing up the stack
		}
	}
	return def
}

func (p *Properties) Int(def int, key string) int {
	if v := p.Get(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func (p *Properties) Watch() func() {
	if p.close != nil {
		return p.close
	}
	slog.Info(fmt.Sprintf("Watching %s for changes", p.filename))
	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Warn("Failed to create watcher for properties file: " + err.Error())
	}

	_ = watcher.Add(path.Dir(p.filename))

	go func() {

		for e := range watcher.Events {
			if e.Name == p.filename && e.Op != fsnotify.Chmod {
				if err := p.LoadFile(p.filename); err != nil {
					fmt.Printf("Error reloading %s: %s\n", p.filename, err)
				}

			}
		}
	}()
	return func() {
		_ = watcher.Close()
	}
}
