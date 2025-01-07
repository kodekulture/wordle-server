package config

import "github.com/escalopa/goconfig"

var cfg = goconfig.New()

// Get ...
func Get(key string) string {
	return cfg.Get(key)
}

// GetOrDefault ...
func GetOrDefault(key, def string) string {
	env := cfg.Get(key)
	if env != "" {
		return env
	}
	return def
}
