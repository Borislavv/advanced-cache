package prometheusconifg

type Configurator interface {
	IsPrometheusMetricsEnabled() bool
}

type Metrics struct {
	// IsEnabledPrometheusMetrics defines wether prometheus metrics will be enabled on the server (basic metrics by default).
	IsEnabledPrometheusMetrics bool `envconfig:"IS_PROMETHEUS_METRICS_ENABLED" mapstructure:"IS_PROMETHEUS_METRICS_ENABLED" default:"true"`
}

func (c Metrics) IsPrometheusMetricsEnabled() bool {
	return c.IsEnabledPrometheusMetrics
}
