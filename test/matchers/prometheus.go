package matchers

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/bsm/gomega/gcustom"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/samber/lo"
)

func MatchCounter(val int64, labels ...string) gcustom.CustomGomegaMatcher {
	return gcustom.MakeMatcher(func(metric string) (bool, error) {

		value, err := getMetric(metric, labels...)
		if err != nil {
			return false, err
		}

		if value.Gauge != nil {

			if value.Gauge.Value == nil {
				return false, fmt.Errorf("no metric exported for %s ", metric)
			}

			if v := int64(*value.Gauge.Value); v == val {
				return true, nil
			} else {
				return false, fmt.Errorf("expected %d, got %d", val, v)
			}

		} else if value.Counter != nil {

			if value.Counter.Value == nil {
				return false, fmt.Errorf("no metric exported for %s ", metric)
			}
			if v := int64(*value.Counter.Value); v == val {
				return true, nil
			} else {
				return false, fmt.Errorf("expected %d, got %d", val, v)
			}

		} else if value.Histogram != nil {

			if strings.HasSuffix(metric, "_count") {
				if value.Histogram.SampleCount == nil {
					return false, fmt.Errorf("no metric exported for %s ", metric)
				}

				if v := int64(*value.Histogram.SampleCount); v == val {
					return true, nil
				} else {
					return false, fmt.Errorf("expected %d, got %d", val, v)
				}

			} else if strings.HasSuffix(metric, "_sum") {
				if value.Histogram.SampleSum == nil {
					return false, fmt.Errorf("no metric exported for %s ", metric)
				}
				if v := int64(*value.Histogram.SampleSum); v == val {
					return true, nil
				} else {
					return false, fmt.Errorf("expected %d, got %d", val, v)
				}
			} else {
				return false, fmt.Errorf("unknown histogram metric: %v", metric)
			}

		} else {
			return false, fmt.Errorf("%s is not a counter or a guage", metric)
		}

	})
}

func DumpMetrics(prefix string) string {
	prom := prometheus.DefaultGatherer
	_metrics, err := prom.Gather()
	if err != nil {
		return err.Error()
	}

	out := bytes.NewBuffer(make([]byte, 0, 1024))
	for _, v := range lo.Filter(_metrics, func(i *io_prometheus_client.MetricFamily, _ int) bool {
		return strings.HasPrefix(lo.FromPtr(i.Name), prefix)
	}) {
		if _, err := expfmt.MetricFamilyToText(out, v); err != nil {
			return err.Error()
		}

	}
	return out.String()
}

func getMetric(name string, labels ...string) (*io_prometheus_client.Metric, error) {
	prom := prometheus.DefaultGatherer
	_metrics, err := prom.Gather()
	if err != nil {
		return nil, err
	}
	_metrics = lo.Filter(_metrics, func(i *io_prometheus_client.MetricFamily, _ int) bool {
		return !strings.HasPrefix(lo.FromPtr(i.Name), "go_")
	})

	labelMap := map[string]string{}

	for i := 0; i < len(labels)-1; i = i + 2 {
		labelMap[labels[i]] = labels[i+1]
	}

	for _, i := range _metrics {
		if !strings.HasPrefix(name, lo.FromPtr(i.Name)) {
			continue
		}

	outer:
		for _, _metric := range i.Metric {
			if len(labels) == 0 {
				return _metric, nil
			}
			metricLabels := map[string]string{}
			for _, labelPair := range _metric.Label {
				metricLabels[labelPair.GetName()] = labelPair.GetValue()
			}
			for k, v := range labelMap {
				if metricLabels[k] != v {
					log.Printf("%s %s: %s != %s", name, k, v, metricLabels[k])
					continue outer
				}
			}
			return _metric, nil
		}
	}

	return nil, fmt.Errorf("metric %s{%v} not found", name, labelMap)
}
