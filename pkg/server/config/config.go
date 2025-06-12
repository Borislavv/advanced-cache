package config

import (
	"time"
)

type Configurator interface {
	GetHttpServerName() string
	GetHttpServerPort() string
	GetHttpServerShutDownTimeout() time.Duration
	GetHttpServerRequestTimeout() time.Duration
	IsPrometheusMetricsEnabled() bool
}

type HttpServer struct {
	// ServerName is a name of the shared server.
	ServerName string `envconfig:"SERVER_NAME" mapstructure:"SERVER_NAME" default:"DefLang"`
	// ServerPort is a port for shared server (endpoints like a /probe for k8s).
	ServerPort string `envconfig:"SERVER_PORT" mapstructure:"SERVER_PORT" default:":8020"`
	// ServerShutDownTimeout is a duration value before the server will be closed forcefully.
	ServerShutDownTimeout time.Duration `envconfig:"SERVER_SHUTDOWN_TIMEOUT" mapstructure:"SERVER_SHUTDOWN_TIMEOUT" default:"5s"`
	// ServerRequestTimeout is a timeout value for close request forcefully.
	ServerRequestTimeout time.Duration `envconfig:"SERVER_REQUEST_TIMEOUT" mapstructure:"SERVER_REQUEST_TIMEOUT" default:"1m"`
	// IsEnabledPrometheusMetrics defines wether prometheus metrics will be enabled on the server (basic metrics by default).
	IsEnabledPrometheusMetrics bool `envconfig:"IS_PROMETHEUS_METRICS_ENABLED" mapstructure:"IS_PROMETHEUS_METRICS_ENABLED" default:"true"`
}

func (c HttpServer) GetHttpServerName() string {
	return c.ServerName
}

func (c HttpServer) GetHttpServerPort() string {
	return c.ServerPort
}

func (c HttpServer) GetHttpServerShutDownTimeout() time.Duration {
	return c.ServerShutDownTimeout
}

func (c HttpServer) GetHttpServerRequestTimeout() time.Duration {
	return c.ServerRequestTimeout
}

func (c HttpServer) IsPrometheusMetricsEnabled() bool {
	return c.IsEnabledPrometheusMetrics
}
