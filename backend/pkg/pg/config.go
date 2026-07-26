package pg

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
)

// Config holds the PostgreSQL connection configuration loaded from environment variables.
type Config struct {
	Host        string `env:"HOST"`
	Port        int    `env:"PORT"`
	User        string `env:"USER"`
	Password    string `env:"PASSWORD"`
	Database    string `env:"DATABASE"`
	SSLMode     string `env:"SSLMODE"`
	AutoMigrate bool   `env:"AUTO_MIGRATE"`
}

// Validate returns an error if the configuration is invalid.
func (c Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("HOST must not be empty")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT %d is outside range [1, 65535]", c.Port)
	}
	if c.User == "" {
		return fmt.Errorf("USER must not be empty")
	}
	if c.Database == "" {
		return fmt.Errorf("DATABASE must not be empty")
	}
	switch c.SSLMode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("SSLMODE %q is not a valid ssl mode", c.SSLMode)
	}
	return nil
}

// DSN builds the PostgreSQL connection string from the configuration.
func (c Config) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Password),
		Host:   net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
		Path:   c.Database,
	}
	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}
