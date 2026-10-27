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
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/flanksource/commons/timeinterval"
)

const businessHoursKey = "business_hours"

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

// Properties represents a thread-safe key-value store for application configuration.
// It supports loading from files, dynamic updates, file watching, and change notifications.
type Properties struct {
	m         map[string]string   // The property map
	metadata  sync.Map            // Typed accessor declarations
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

func (p *Properties) Unset(key string) {
	p.lock.Lock()
	delete(p.m, key)
	p.lock.Unlock()
	p.notify()
}

func (p *Properties) notify() {
	for _, listener := range p.listeners {
		listener(p)
	}
}

func (p *Properties) GetAll() map[string]string {
	p.lock.RLock()
	m := maps.Clone(p.m)
	p.lock.RUnlock()
	//command line properties take priority
	maps.Copy(m, commandline.snapshot())
	return m
}

func (p *Properties) Get(key string) string {
	//command line properties take priority
	if v, ok := commandline.get(key); ok {
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
		p.Update(commandline.snapshot())
		return nil
	} else if err != nil {
		return err
	}
	defer file.Close()
	p.filename = filename

	if p.close == nil {
		p.Watch()
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

	maps.Copy(props, commandline.snapshot())
	p.Update(props)

	return nil
}

func RegisterListener(fn func(*Properties)) {
	Global.RegisterListener(fn)
}

func Set(key string, value any) {
	Global.Set(key, value)
}

func Unset(key string) { Global.Unset(key) }

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

func Bytes(def int, key string) int {
	return Global.Bytes(def, key)
}

func Choice(def string, options []string, keys ...string) string {
	return Global.Choice(def, options, keys...)
}

func LogLevel(def string, options []string, keys ...string) string {
	return Global.LogLevel(def, options, keys...)
}

func BusinessHours() (timeinterval.TimeIntervals, error) {
	hours, err := Global.TimeIntervals(businessHoursKey)
	if err != nil {
		return nil, err
	}

	if len(hours) == 0 {
		return []timeinterval.TimeInterval{{
			Times: []timeinterval.TimeRange{
				{
					StartMinute: 540,  // 9am
					EndMinute:   1020, // 5pm
				},
			},
			Weekdays: []timeinterval.WeekdayRange{
				{InclusiveRange: timeinterval.InclusiveRange{
					Begin: 1, // Monday
					End:   5, // Friday
				}},
			},
		}}, nil
	}

	return hours, nil
}

func TimeIntervals(keys ...string) (timeinterval.TimeIntervals, error) {
	return Global.TimeIntervals(keys...)
}

func (p *Properties) On(def bool, keys ...string) bool {
	p.registerDefault(PropertyTypeBool, strconv.FormatBool(def), keys...)
	for _, key := range keys {
		if v := p.Get(key); v != "" {
			return strings.ToLower(v) == "true"
		}
	}
	return def
}

func (p *Properties) String(def string, keys ...string) string {
	p.registerDefault(PropertyTypeString, def, keys...)
	for _, key := range keys {
		if v := p.Get(key); v != "" {
			return v
		}
	}
	return def
}

func (p *Properties) Choice(def string, options []string, keys ...string) string {
	p.registerChoice(PropertyTypeChoice, def, options, keys...)
	for _, key := range keys {
		if v := p.Get(key); v != "" {
			return v
		}
	}
	return def
}

func (p *Properties) LogLevel(def string, options []string, keys ...string) string {
	p.registerChoice(PropertyTypeLogLevel, def, options, keys...)
	for _, key := range keys {
		if v := p.Get(key); v != "" {
			return v
		}
	}
	return def
}

func (p *Properties) Duration(def time.Duration, keys ...string) time.Duration {
	p.registerDefault(PropertyTypeDuration, def.String(), keys...)
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
	p.registerDefault(PropertyTypeInt, strconv.Itoa(def), key)
	if v := p.Get(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

// Bytes returns the byte size configured under key, falling back to def when
// the property is unset or does not parse. Values use the ParseBytes syntax: a
// decimal integer with an optional size suffix, e.g. "1048576", "1MiB" or
// "4MB".
// Only unset and unparseable values fall back to def; a value that parses to 0
// or less is returned as-is so callers can use it to disable a cap. Callers
// that require a positive limit must reject non-positive results themselves,
// as logger.HTTPLogResponseBodyLength does.
func (p *Properties) Bytes(def int, key string) int {
	p.registerDefault(PropertyTypeBytes, strconv.Itoa(def), key)
	if v := p.Get(key); v != "" {
		if parsed, err := ParseBytes(v); err == nil && int64(int(parsed)) == parsed {
			return int(parsed)
		}
	}
	return def
}

// TimeIntervals returns the parsed time intervals from the specified property keys.
// Properties are looked up under "time_interval.<key>" prefix.
// If no keys are provided, defaults to "business_hours".
// The property value should be a JSON array in alertmanager time interval format.
// Example: [{"weekdays":["monday:friday"],"times":[{"start_time":"09:00","end_time":"17:00"}]}]
func (p *Properties) TimeIntervals(keys ...string) (timeinterval.TimeIntervals, error) {
	if len(keys) == 0 {
		keys = []string{businessHoursKey}
	}
	propertyKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		propertyKeys = append(propertyKeys, "time_interval."+key)
	}
	p.registerType(PropertyTypeTimeIntervals, propertyKeys...)

	var output []timeinterval.TimeInterval
	for _, key := range keys {
		propKey := "time_interval." + key
		v := p.Get(propKey)
		if v == "" {
			continue
		}

		var intervals []timeinterval.TimeInterval
		if err := json.Unmarshal([]byte(v), &intervals); err != nil {
			return nil, fmt.Errorf("failed to parse time interval from property %s: %w", propKey, err)
		}
		output = append(output, intervals...)
	}

	return output, nil
}

func (p *Properties) Watch() func() {
	if p.close != nil {
		return p.close
	}
	slog.Info(fmt.Sprintf("Watching %s for changes", p.filename))
	info, err := os.Stat(p.filename)
	if err != nil {
		panic(fmt.Errorf("watch properties %s: %w", p.filename, err))
	}
	filename := p.filename
	stopped := make(chan struct{})
	var once sync.Once
	p.close = func() { once.Do(func() { close(stopped) }) }
	go func() {
		ticker := time.NewTicker(WatchInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stopped:
				return
			case <-ticker.C:
				current, err := os.Stat(filename)
				if err != nil {
					slog.Error("stat properties file", "file", filename, "error", err)
					continue
				}
				if current.ModTime().Equal(info.ModTime()) && current.Size() == info.Size() && os.SameFile(info, current) {
					continue
				}
				if err := p.LoadFile(filename); err != nil {
					slog.Error("reload properties file", "file", filename, "error", err)
					continue
				}
				info = current
			}
		}
	}()
	return p.close
}

// WatchInterval is the interval used to poll a loaded properties file's mtime.
const WatchInterval = 100 * time.Millisecond
