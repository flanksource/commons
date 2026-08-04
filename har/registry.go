package har

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/properties"
)

// PropertyPrefix is the namespace the registry resolves its properties under.
const PropertyPrefix = "http."

// Registry turns -P properties into HAR capture: it resolves the output path
// and level per feature, owns one collector per output file, and writes them
// all on Flush.
//
// Properties are looked up per-feature first, then globally:
//
//	http.<feature>.har        / http.har        output path; unset disables capture
//	http.<feature>.har.level  / http.har.level  "full" (default) or "metadata"
//	http.har.sensitive                          capture credentials verbatim
//	http.har.maxBodySize                        per-body capture cap
//
// Collectors are deduplicated by absolute path, so several features writing to
// the same file share one archive.
type Registry struct {
	prefix     string
	log        logger.Logger
	collectors sync.Map // absolute path -> *Collector
}

// NewRegistry returns a registry that announces capture as it is enabled and
// reports flush results on log. Pass nil for the shared "har" logger, whose
// level can be raised on its own with -Plog.level.har=debug.
func NewRegistry(log logger.Logger) *Registry {
	if log == nil {
		log = logger.GetLogger("har")
	}
	return &Registry{prefix: PropertyPrefix, log: log}
}

// For reports the collector, absolute output path and level configured for
// feature. A nil collector means capture is off. The shape matches
// http.CommonsHTTPContext's HARFor apart from the error, which reports an
// unusable http.har.level rather than silently capturing the wrong thing.
func (r *Registry) For(feature string) (*Collector, string, Level, error) {
	path := r.lookup(feature, "har")
	if path == "" {
		return nil, "", Disabled, nil
	}

	level, err := ParseLevel(r.lookup(feature, "har.level"), Full)
	if err != nil {
		return nil, "", Disabled, fmt.Errorf("%s%s: %w", r.prefix, "har.level", err)
	}
	if level == Disabled {
		return nil, "", Disabled, nil
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, "", Disabled, fmt.Errorf("resolve HAR path %q: %w", path, err)
	}
	collector, created := r.collectorFor(abs)
	if created {
		r.log.Infof("capturing HAR to %s (%s)", abs, describeMode(collector.Config, level))
	}
	return collector, abs, level, nil
}

// describeMode renders the capture mode for the announcement, e.g.
// "level=full" or "level=full, sensitive".
func describeMode(cfg HARConfig, level Level) string {
	mode := "level=" + level.String()
	if cfg.CaptureSensitive {
		mode += ", sensitive"
	}
	return mode
}

// Transport wraps base with the capture middleware configured for feature, or
// returns base unchanged when capture is off.
func (r *Registry) Transport(feature string, base http.RoundTripper) (http.RoundTripper, error) {
	collector, _, level, err := r.For(feature)
	if err != nil || collector == nil {
		return base, err
	}
	if base == nil {
		base = http.DefaultTransport
	}

	if level == Metadata {
		return NewMetadataMiddleware(collector.Config, collector.Add)(base), nil
	}
	return collector.Middleware()(base), nil
}

// Flush writes every collector to its file. Collectors are kept afterwards, so
// a second call rewrites the same files rather than losing entries.
func (r *Registry) Flush() error {
	var errs []error
	r.collectors.Range(func(key, value any) bool {
		path, collector := key.(string), value.(*Collector)
		if err := WriteFile(collector, path); err != nil {
			errs = append(errs, fmt.Errorf("write HAR %s: %w", path, err))
			return true
		}
		r.log.Infof("wrote HAR %s (%d entries)", path, len(collector.Entries()))
		if collector.Config.CaptureSensitive {
			r.log.Warnf("%s contains unredacted credentials (%s=true)", path, SensitiveProperty)
		}
		return true
	})
	return errors.Join(errs...)
}

// collectorFor returns the collector for absPath, reporting whether this call
// created it. Only the creating call announces, so one archive logs one line no
// matter how many features resolve to it.
func (r *Registry) collectorFor(absPath string) (*Collector, bool) {
	if existing, ok := r.collectors.Load(absPath); ok {
		return existing.(*Collector), false
	}
	actual, loaded := r.collectors.LoadOrStore(absPath, NewCollector(DefaultConfig()))
	return actual.(*Collector), !loaded
}

func (r *Registry) lookup(feature, suffix string) string {
	if feature != "" {
		if v := properties.String("", r.prefix+feature+"."+suffix); v != "" {
			return v
		}
	}
	return properties.String("", r.prefix+suffix)
}
