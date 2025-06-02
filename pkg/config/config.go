package config

type Config struct {
	Storage    `mapstructure:",squash"`
	Response   `mapstructure:",squash"`
	Repository `mapstructure:",squash"`
}
