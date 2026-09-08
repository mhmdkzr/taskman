package config

import (
	"fmt"
	"time"
)

type ServerConfig struct {
	BindAddr        string        `env:"BIND_ADDR"`
	BasePath        string        `env:"BASE_PATH"`
	Timeout         time.Duration `env:"TIMEOUT"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT"`
}

func (c ServerConfig) validate() error {
	if c.BindAddr == "" {
		return fmt.Errorf("BIND_ADDR must not be empty")
	}
	return nil
}
