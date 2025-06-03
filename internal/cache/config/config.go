package config

import (
	cacheConfig "github.com/Borislavv/traefik-http-cache-plugin/pkg/config"
	"gitlab.xbet.lan/v3group/backend/packages/go/httpserver/pkg/httpserver/config"
)

type Config struct {
	config.HttpServer  `mapstructure:",squash"`
	cacheConfig.Config `mapstructure:",squash"`
}
