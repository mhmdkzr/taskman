package config

import (
	"fmt"
	"time"
)

// BrowserConfig configures the browser automation agent tools backed by go-rod.
// The tools are enabled by default; set AGENT_BROWSER_ENABLED=false to disable.
type BrowserConfig struct {
	Enabled  bool          `env:"ENABLED"  envDefault:"true"`
	Headless bool          `env:"HEADLESS" envDefault:"true"`
	Bin      string        `env:"BIN"      envDefault:""`
	CDPURL   string        `env:"CDP_URL"  envDefault:""`
	Timeout  time.Duration `env:"TIMEOUT"  envDefault:"60s"`
}

func (c BrowserConfig) validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("TIMEOUT must be positive")
	}
	return nil
}