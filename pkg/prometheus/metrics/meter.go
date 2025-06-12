package metrics

import (
	"errors"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/prometheus/metrics/keyword"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/prometheus/metrics/validator"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

var MetricRegisterErrorMessage = "failed to register metric counter"

type Meter interface {
	IncTotal(path string, method string, status string)
	IncStatus(path string, method string, status string)
	NewResponseTimeTimer(path string, method string) *prometheus.Timer
	FlushResponseTimeTimer(t *prometheus.Timer)
}

type Metrics struct {
	totalRequestsCounter    *prometheus.CounterVec
	totalResponsesCounter   *prometheus.CounterVec
	responseStatusesCounter *prometheus.CounterVec
	responseTimeMsCounter   *prometheus.HistogramVec
}

func New() (*Metrics, error) {
	m := &Metrics{
		totalRequestsCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: keyword.TotalHttpRequestsMetricName,
				Help: "Number of all requests.",
			},
			[]string{"path", "method"},
		),
		totalResponsesCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: keyword.TotalHttpResponsesMetricName,
				Help: "Number of all responses.",
			},
			[]string{"path", "method", "status"},
		),
		responseStatusesCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: keyword.HttpResponseStatusesMetricName,
				Help: "Status of HTTP response",
			},
			[]string{"path", "method", "status"},
		),
		responseTimeMsCounter: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: keyword.HttpResponseTimeMsMetricName,
			Help: "Duration of HTTP requests.",
		}, []string{"path", "method"}),
	}

	if err := prometheus.Register(m.totalRequestsCounter); err != nil {
		log.Err(err).Msg(MetricRegisterErrorMessage)
		return nil, errors.New(MetricRegisterErrorMessage)
	}
	if err := prometheus.Register(m.totalResponsesCounter); err != nil {
		log.Err(err).Msg(MetricRegisterErrorMessage)
		return nil, errors.New(MetricRegisterErrorMessage)
	}
	if err := prometheus.Register(m.responseStatusesCounter); err != nil {
		log.Err(err).Msg(MetricRegisterErrorMessage)
		return nil, errors.New(MetricRegisterErrorMessage)
	}
	if err := prometheus.Register(m.responseTimeMsCounter); err != nil {
		log.Err(err).Msg(MetricRegisterErrorMessage)
		return nil, errors.New(MetricRegisterErrorMessage)
	}

	return m, nil
}

// IncTotal method is increments request/response total counters and depends on
// *status* argument (numeric or empty string available).
// If the *status* argument is empty string then will be used request_counter,
// in other way will be used response_counter.
func (m *Metrics) IncTotal(path string, method string, status string) {
	if status != "" {
		if err := validator.ValidateStrStatusCode(status); err != nil {
			panic(err)
		}
		m.totalResponsesCounter.With(
			prometheus.Labels{
				"path":   path,
				"method": method,
				"status": status,
			},
		).Inc()
		return
	}
	m.totalRequestsCounter.With(
		prometheus.Labels{
			"path":   path,
			"method": method,
		},
	).Inc()
}

func (m *Metrics) IncStatus(path string, method string, status string) {
	if err := validator.ValidateStrStatusCode(status); err != nil {
		panic(err)
	}

	m.responseStatusesCounter.With(
		prometheus.Labels{
			"path":   path,
			"method": method,
			"status": status,
		},
	).Inc()
}

func (m *Metrics) NewResponseTimeTimer(path string, method string) *prometheus.Timer {
	return prometheus.NewTimer(m.responseTimeMsCounter.WithLabelValues(path, method))
}

func (m *Metrics) FlushResponseTimeTimer(t *prometheus.Timer) {
	t.ObserveDuration()
}
