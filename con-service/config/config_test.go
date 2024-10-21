package config

import (
	"testing"
)

func TestInitialize(t *testing.T) {
	config := Initialize()
	if config == nil {
		t.Error("Expected config to be initialized, but it was nil")
	}

	if config.Service.Name == "" {
		t.Error("Expected Name to be set, but it was empty")
	}

	if config.Service.Port == "" {
		t.Error("Expected Port to be set, but it was empty")
	}
}
