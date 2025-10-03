// Package properties provides a thread-safe properties management system with
// file loading, hot-reloading, and change notification support.
//
// The package offers a simple key-value store for application configuration
// with support for loading from properties files, command-line overrides,
// and dynamic updates with file watching.
//
// Key features:
//   - Load properties from files (key=value format)
//   - Command-line property overrides via -P flag
//   - File watching with automatic hot-reloading
//   - Thread-safe operations with read-write locks
//   - Change listeners for reactive configuration
//   - Type-safe accessors for common data types
//
// Basic usage:
//
//	// Load properties from file
//	err := properties.LoadFile("app.properties")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Get property values with defaults
//	host := properties.String("localhost", "server.host")
//	port := properties.Int(8080, "server.port")
//	debug := properties.On(false, "debug.enabled", "debug")
//	timeout := properties.Duration(30*time.Second, "request.timeout")
//
//	// Set properties dynamically
//	properties.Set("api.key", "secret-key")
//
//	// Register change listener
//	properties.RegisterListener(func(p *properties.Properties) {
//		log.Println("Properties updated")
//	})
//
// Command-line usage:
//
//	./app -P db.host=localhost -P db.port=5432
//
// Properties file format:
//
//	# Comments start with #
//	server.host=localhost
//	server.port=8080
//	debug.enabled=true
//	request.timeout=30s
//
// The package maintains a global instance for convenience, but you can also
// create isolated Properties instances for different configuration contexts.
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

// Global is the default properties instance used by package-level functions.
// It's automatically initialized and ready to use.
var Global = &Properties{
	m: make(map[string]string),
}

// LoadFile is a convenience function that loads properties from a file
// into the global properties instance. It's equivalent to Global.LoadFile(filename).
var LoadFile = func(filename string) error {
	return Global.LoadFile(filename)
}

// BindFlags binds the -P/--properties flag to the given flag set, allowing
// properties to be set via command line.
//
// Example:
//
//	flags := pflag.NewFlagSet("app", pflag.ContinueOnError)
//	properties.BindFlags(flags)
//	flags.Parse(os.Args[1:])
//	// Now you can use: ./app -P key1=value1 -P key2=value2
func BindFlags(flags *pflag.FlagSet) {
	flags.StringToStringVarP(&commandlineProperties, "properties", "P", nil, "System properties")
}

// Properties represents a thread-safe key-value store for application configuration.
// It supports loading from files, dynamic updates, file watching, and change notifications.
type Properties struct {
	m         map[string]string   // The property map
	filename  string              // Currently loaded file
	listeners []func(*Properties) // Change listeners
	lock      sync.RWMutex        // Protects concurrent access
	close     func()              // Cleanup function for file watcher
	Reload    func()              // Function to manually trigger reload
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
	//command line properties take priority
	for k, v := range commandlineProperties {
		m[k] = v
	}
	return m
}

func (p *Properties) Get(key string) string {
	//command line properties take priority
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
