package config

import (
	cacheConfig "github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"github.com/Borislavv/traefik-http-cache-plugin/pkg/server/config"
)

type Config struct {
	config.HttpServer  `mapstructure:",squash"`
	cacheConfig.Config `mapstructure:",squash"`
}
