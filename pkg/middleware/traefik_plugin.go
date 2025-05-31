package middleware

import (
	"context"
	"net/http"
)

type Config struct {
}

func CreateConfig() *Config {
	return &Config{}
}

type Plugin struct {
	next   http.Handler
	name   string
	config *Config
}

func New(ctx context.Context, next http.Handler, config *Config, name string) http.Handler {
	return &Plugin{
		next:   next,
		name:   name,
		config: config,
	}
}

func (l *Plugin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//project := r.URL.Query().Get("project")
	//domain := r.URL.Query().Get("domain")
	//language := r.URL.Query().Get("language")
	//choice := r.URL.Query().Get("choice")
}
