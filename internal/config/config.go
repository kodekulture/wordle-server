package config

import "github.com/escalopa/goconfig"

var cfg = goconfig.New()

// Get ...
func Get(key string) string {
	return cfg.Get(key)
}
