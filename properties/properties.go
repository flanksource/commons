package properties

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
)

var Global = &Properties{
	m: make(map[string]string),
}

type Properties struct {
	m         map[string]string
	listeners []func(*Properties)
	lock      sync.RWMutex
}

func (p *Properties) RegisterListener(fn func(*Properties)) {
	p.listeners = append(p.listeners, fn)
}

func (p *Properties) Set(key string, value any) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.m[key] = fmt.Sprintf("%v", value)
	p.notify()
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
	return m
}

func (p *Properties) Get(key string) string {
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

func LoadFile(filename string) error {
	return Global.LoadFile(filename)
}

func (p *Properties) LoadFile(filename string) error {
	file, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		slog.Debug(fmt.Sprintf("%s does not exist", filename))
		return nil
	} else if err != nil {
		return err
	}
	defer file.Close()
	slog.Info(fmt.Sprintf("loading properties from %s", filename))
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

func (p *Properties) Int(def int, key string) int {
	if v := p.Get(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
