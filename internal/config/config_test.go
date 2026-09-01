package config

import "testing"

func TestNATSConfigValidate(t *testing.T) {
	if err := (NATSConfig{URL: "nats://127.0.0.1:4222"}).validate(); err != nil {
		t.Fatalf("validate valid config: %v", err)
	}
	if err := (NATSConfig{URL: ""}).validate(); err == nil {
		t.Fatal("validate config with missing URL succeeded")
	}
}

func TestServerConfigValidate(t *testing.T) {
	if err := (ServerConfig{BindAddr: "127.0.0.1:8080"}).validate(); err != nil {
		t.Fatalf("validate valid config: %v", err)
	}
	if err := (ServerConfig{BindAddr: ""}).validate(); err == nil {
		t.Fatal("validate config with missing BIND_ADDR succeeded")
	}
}
