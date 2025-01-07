package config

import (
	"github.com/escalopa/goconfig"
	zlog "github.com/rs/zerolog/log"
)

var cfg = goconfig.New()

// Get ...
func Get(key string) string {
	return cfg.Get(key)
}

// GetOrDefault ...
func GetOrDefault[T any](key string, def T, fn func(envValue string) (T, error)) T {
	env := cfg.Get(key)
	if env == "" {
		return def
	}
	conv, err := fn(env)
	if err != nil {
		zlog.Warn().Msgf("Tried to convert value from env variable %s to type %T: got error: %v. Falling back to default value: %v", key, env, err, def)
		return def
	}
	return conv
}
