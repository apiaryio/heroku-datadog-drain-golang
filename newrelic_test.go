package main

import (
	"os"
	"testing"
)

func TestBuildNewRelicConfig(t *testing.T) {
	os.Setenv("NEWRELIC_LICENSE_KEY", "foo")
	defer os.Setenv("NEWRELIC_LICENSE_KEY", "")

	t.Run("When Newrelic tracing is disabled", func(t *testing.T) {
		os.Setenv("NEWRELIC_ENABLED", "false")
		defer os.Setenv("NEWRELIC_ENABLED", "")

		config := BuildNewRelicConfig()

		if config.Enabled != false {
			t.Error("expected", false, "got", config.Enabled)
		}
	})

	t.Run("When Newrelic license key is not set", func(t *testing.T) {
		os.Setenv("NEWRELIC_LICENSE_KEY", "")
		defer os.Setenv("NEWRELIC_LICENSE_KEY", "foo")

		config := BuildNewRelicConfig()

		if config.Enabled != false {
			t.Error("expected", false, "got", config.Enabled)
		}
	})

	t.Run("When Newrelic tracing is enabled", func(t *testing.T) {
		config := BuildNewRelicConfig()

		if config.Enabled != true {
			t.Error("expected", true, "got", config.Enabled)
		}
	})
}
