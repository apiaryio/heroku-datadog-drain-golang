package main

import (
	"testing"
)

func TestBuildNewRelicConfig(t *testing.T) {
	cases := []struct {
		enabled, licenseKey string
		expected            bool
	}{
		{"false", "roo", false},
		{"", "foo", true},
		{"", "", false},
	}

	for _, c := range cases {
		config := BuildNewRelicConfig(c.enabled, c.licenseKey)

		if config.Enabled != c.expected {
			t.Error("expected", c.expected, "got", config.Enabled)
		}
	}
}
