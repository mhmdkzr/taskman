package config

import "fmt"

type SQLiteConfig struct {
	Path        string `env:"PATH"         envDefault:"loop.db"`
	AutoMigrate bool   `env:"AUTO_MIGRATE" envDefault:"true"`
}

func (c SQLiteConfig) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("PATH must not be empty")
	}
	return nil
}
