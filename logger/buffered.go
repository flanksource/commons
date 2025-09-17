package logger

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// BufferedLogEntry represents a single buffered log message
type BufferedLogEntry struct {
	Message string
	Time    time.Time
	Level   LogLevel
}

// BufferedLogger implements Logger interface with in-memory log storage
type BufferedLogger struct {
	mu             sync.RWMutex
	logsByLevel    map[LogLevel][]BufferedLogEntry
	maxLogsByLevel map[LogLevel]int
	logLevel       LogLevel
}

// NewBufferedLogger creates a new BufferedLogger with default retention strategy
func NewBufferedLogger(maxLogs int) *BufferedLogger {
	return NewBufferedLoggerWithRetention(getDefaultRetentionConfig(maxLogs))
}

// RetentionConfig defines retention limits for each log level
type RetentionConfig map[LogLevel]int

// NewBufferedLoggerWithRetention creates a new BufferedLogger with custom retention config
func NewBufferedLoggerWithRetention(config RetentionConfig) *BufferedLogger {
	logsByLevel := make(map[LogLevel][]BufferedLogEntry)
	maxLogsByLevel := make(map[LogLevel]int)

	// Initialize buffers for each configured level
	for level, maxCount := range config {
		logsByLevel[level] = make([]BufferedLogEntry, 0, maxCount)
		maxLogsByLevel[level] = maxCount
	}

	return &BufferedLogger{
		logsByLevel:    logsByLevel,
		maxLogsByLevel: maxLogsByLevel,
		logLevel:       Info,
	}
}

// getDefaultRetentionConfig returns default retention limits based on total maxLogs
func getDefaultRetentionConfig(maxLogs int) RetentionConfig {
	if maxLogs <= 0 {
		maxLogs = 50 // Default total
	}

	// Base retention strategy
	baseConfig := map[LogLevel]int{
		Fatal: maxLogs * 20 / 100, // 20% for Fatal
		Error: maxLogs * 25 / 100, // 25% for Error
		Warn:  maxLogs * 20 / 100, // 20% for Warn
		Info:  maxLogs * 20 / 100, // 20% for Info
		Debug: maxLogs * 10 / 100, // 10% for Debug
		Trace: maxLogs * 5 / 100,  // 5% for Trace
	}

	// Ensure minimum of 1 for each level
	for level := range baseConfig {
		if baseConfig[level] < 1 {
			baseConfig[level] = 1
		}
	}

	return baseConfig
}

// appendLog adds a log entry to the appropriate level buffer
func (b *BufferedLogger) appendLog(level LogLevel, format string, args ...interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	entry := BufferedLogEntry{
		Message: fmt.Sprintf(format, args...),
		Time:    time.Now(),
		Level:   level,
	}

	// Get or create buffer for this level
	if _, exists := b.logsByLevel[level]; !exists {
		// Initialize with default retention if not configured
		maxCount := 10 // Default
		if configured, ok := b.maxLogsByLevel[level]; ok {
			maxCount = configured
		}
		b.logsByLevel[level] = make([]BufferedLogEntry, 0, maxCount)
		b.maxLogsByLevel[level] = maxCount
	}

	// Add to appropriate level buffer
	b.logsByLevel[level] = append(b.logsByLevel[level], entry)

	// Trim if we exceed max logs for this level
	maxForLevel := b.maxLogsByLevel[level]
	if len(b.logsByLevel[level]) > maxForLevel {
		b.logsByLevel[level] = b.logsByLevel[level][len(b.logsByLevel[level])-maxForLevel:]
	}
}

// GetLogs returns a copy of all buffered log entries, sorted by timestamp
func (b *BufferedLogger) GetLogs() []BufferedLogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Collect all entries from all level buffers
	var allEntries []BufferedLogEntry
	for _, levelEntries := range b.logsByLevel {
		allEntries = append(allEntries, levelEntries...)
	}

	// Sort by timestamp (oldest first)
	for i := 0; i < len(allEntries)-1; i++ {
		for j := i + 1; j < len(allEntries); j++ {
			if allEntries[i].Time.After(allEntries[j].Time) {
				allEntries[i], allEntries[j] = allEntries[j], allEntries[i]
			}
		}
	}

	return allEntries
}

// ClearLogs clears all buffered log entries
func (b *BufferedLogger) ClearLogs() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for level := range b.logsByLevel {
		b.logsByLevel[level] = b.logsByLevel[level][:0]
	}
}

// GetLogsByLevel returns buffered log entries for a specific level
func (b *BufferedLogger) GetLogsByLevel(level LogLevel) []BufferedLogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if entries, exists := b.logsByLevel[level]; exists {
		result := make([]BufferedLogEntry, len(entries))
		copy(result, entries)
		return result
	}
	return []BufferedLogEntry{}
}

// SetRetentionPolicy updates the retention limits for log levels
func (b *BufferedLogger) SetRetentionPolicy(config RetentionConfig) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for level, maxCount := range config {
		b.maxLogsByLevel[level] = maxCount

		// If buffer already exists and exceeds new limit, trim it
		if entries, exists := b.logsByLevel[level]; exists && len(entries) > maxCount {
			b.logsByLevel[level] = entries[len(entries)-maxCount:]
		}
	}
}

// GetRetentionPolicy returns current retention limits
func (b *BufferedLogger) GetRetentionPolicy() RetentionConfig {
	b.mu.RLock()
	defer b.mu.RUnlock()

	config := make(RetentionConfig)
	for level, maxCount := range b.maxLogsByLevel {
		config[level] = maxCount
	}
	return config
}

// ScaleRetentionByLogLevel adjusts retention based on current log level
// Higher verbosity = more retention for all levels
func (b *BufferedLogger) ScaleRetentionByLogLevel() {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Calculate scaling factor based on current log level
	var scaleFactor float64 = 1.0
	switch {
	case b.logLevel >= Trace4:
		scaleFactor = 3.0  // Very verbose, keep lots of logs
	case b.logLevel >= Trace2:
		scaleFactor = 2.5
	case b.logLevel >= Trace:
		scaleFactor = 2.0
	case b.logLevel >= Debug:
		scaleFactor = 1.5
	default:
		scaleFactor = 1.0  // Base retention for Info and below
	}

	// Apply scaling to all configured levels
	baseConfig := getDefaultRetentionConfig(50) // Use base config
	for level, baseCount := range baseConfig {
		newCount := int(float64(baseCount) * scaleFactor)
		if newCount < 1 {
			newCount = 1
		}
		b.maxLogsByLevel[level] = newCount

		// Trim existing buffers if they exceed new limits
		if entries, exists := b.logsByLevel[level]; exists && len(entries) > newCount {
			b.logsByLevel[level] = entries[len(entries)-newCount:]
		}
	}
}

// Warnf logs a warning message
func (b *BufferedLogger) Warnf(format string, args ...interface{}) {
	b.appendLog(Warn, format, args...)
}

// Infof logs an info message
func (b *BufferedLogger) Infof(format string, args ...interface{}) {
	b.appendLog(Info, format, args...)
}

// Errorf logs an error message
func (b *BufferedLogger) Errorf(format string, args ...interface{}) {
	b.appendLog(Error, format, args...)
}

// Debugf logs a debug message
func (b *BufferedLogger) Debugf(format string, args ...interface{}) {
	if b.IsDebugEnabled() {
		b.appendLog(Debug, format, args...)
	}
}

// Tracef logs a trace message
func (b *BufferedLogger) Tracef(format string, args ...interface{}) {
	if b.IsTraceEnabled() {
		b.appendLog(Trace, format, args...)
	}
}

// Fatalf logs a fatal message and panics
func (b *BufferedLogger) Fatalf(format string, args ...interface{}) {
	b.appendLog(Fatal, format, args...)
	panic(fmt.Sprintf(format, args...))
}

// WithValues returns the same logger (noop as requested)
func (b *BufferedLogger) WithValues(keysAndValues ...interface{}) Logger {
	return b
}

// IsTraceEnabled checks if trace level is enabled
func (b *BufferedLogger) IsTraceEnabled() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.logLevel >= Trace
}

// IsDebugEnabled checks if debug level is enabled
func (b *BufferedLogger) IsDebugEnabled() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.logLevel >= Debug
}

// IsLevelEnabled checks if a specific level is enabled
func (b *BufferedLogger) IsLevelEnabled(level LogLevel) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.logLevel >= level
}

// GetLevel returns the current log level
func (b *BufferedLogger) GetLevel() LogLevel {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.logLevel
}

// SetLogLevel sets the log level and automatically scales retention
func (b *BufferedLogger) SetLogLevel(level any) {
	b.mu.Lock()
	defer b.mu.Unlock()

	oldLevel := b.logLevel

	switch v := level.(type) {
	case LogLevel:
		b.logLevel = v
	case int:
		b.logLevel = LogLevel(v)
	case string:
		// Parse string level
		switch v {
		case "trace":
			b.logLevel = Trace
		case "debug":
			b.logLevel = Debug
		case "info":
			b.logLevel = Info
		case "warn":
			b.logLevel = Warn
		case "error":
			b.logLevel = Error
		case "fatal":
			b.logLevel = Fatal
		default:
			b.logLevel = Info
		}
	default:
		b.logLevel = Info
	}

	// Auto-scale retention if log level changed
	if oldLevel != b.logLevel {
		b.mu.Unlock() // Unlock temporarily for ScaleRetentionByLogLevel
		b.ScaleRetentionByLogLevel()
		b.mu.Lock() // Re-lock for defer
	}
}

// SetMinLogLevel sets the minimum log level (same as SetLogLevel for BufferedLogger)
func (b *BufferedLogger) SetMinLogLevel(level any) {
	b.SetLogLevel(level)
}

// V returns a verbose logger
func (b *BufferedLogger) V(level any) Verbose {
	return &bufferedVerbose{
		logger:  b,
		enabled: b.IsLevelEnabled(ParseLevel(b, level)),
		filters: nil,
	}
}

// WithV returns the same logger (for simplicity)
func (b *BufferedLogger) WithV(level any) Logger {
	return b
}

// Named returns the same logger (noop as requested)
func (b *BufferedLogger) Named(name string) Logger {
	return b
}

// WithoutName returns the same logger (noop as requested)
func (b *BufferedLogger) WithoutName() Logger {
	return b
}

// WithSkipReportLevel returns the same logger (noop as requested)
func (b *BufferedLogger) WithSkipReportLevel(i int) Logger {
	return b
}

// GetSlogLogger returns nil (unsupported as requested)
func (b *BufferedLogger) GetSlogLogger() *slog.Logger {
	return nil
}

// bufferedVerbose implements the Verbose interface for BufferedLogger
type bufferedVerbose struct {
	logger  *BufferedLogger
	enabled bool
	filters []string
}

// isFiltered checks if a log line should be filtered out based on filters
func (v *bufferedVerbose) isFiltered(line string) bool {
	if len(strings.TrimSpace(line)) == 0 {
		return true
	}
	for _, filter := range v.filters {
		if strings.Contains(line, filter) {
			return true
		}
	}
	return false
}

// Write implements io.Writer interface
func (v *bufferedVerbose) Write(p []byte) (n int, err error) {
	if !v.enabled {
		return len(p), nil
	}

	for _, line := range strings.Split(string(p), "\n") {
		if v.isFiltered(line) {
			continue
		}
		v.logger.Infof("%s", line)
	}

	return len(p), nil
}

// Infof logs an info message if enabled and not filtered
func (v *bufferedVerbose) Infof(format string, args ...interface{}) {
	if !v.enabled {
		return
	}

	message := fmt.Sprintf(format, args...)
	if !v.isFiltered(message) {
		v.logger.Infof("%s", message)
	}
}

// WithFilter returns a new verbose logger with additional filters
func (v *bufferedVerbose) WithFilter(filters ...string) Verbose {
	newFilters := make([]string, len(v.filters)+len(filters))
	copy(newFilters, v.filters)
	copy(newFilters[len(v.filters):], filters)

	return &bufferedVerbose{
		logger:  v.logger,
		enabled: v.enabled,
		filters: newFilters,
	}
}

// Enabled returns whether this verbose logger is enabled
func (v *bufferedVerbose) Enabled() bool {
	return v.enabled
}
