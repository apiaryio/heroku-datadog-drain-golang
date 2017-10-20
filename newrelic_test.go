package main

import (
	"testing"
)

func TestBuildNewRelicConfig(t *testing.T) {
	t.Run("When Newrelic tracing is disabled", func(t *testing.T) {
		config := BuildNewRelicConfig("false", "foo")

		if config.Enabled != false {
			t.Error("expected", false, "got", config.Enabled)
		}
	})

	t.Run("When Newrelic tracing is enabled", func(t *testing.T) {
		config := BuildNewRelicConfig("", "foo")

		if config.Enabled != true {
			t.Error("expected", true, "got", config.Enabled)
		}
	})

	t.Run("When Newrelic license key is not set", func(t *testing.T) {
		config := BuildNewRelicConfig("", "")

		if config.Enabled != false {
			t.Error("expected", false, "got", config.Enabled)
		}
	})

}
