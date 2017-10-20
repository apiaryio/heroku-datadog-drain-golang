package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/newrelic/go-agent"
)

func BuildNewRelicConfig(enabled, licenseKey string) newrelic.Config {
	config := newrelic.NewConfig("Roo Datadog Bridge", licenseKey)
	config.Enabled = enabled != "false" && licenseKey != ""

	return config
}

func NewNewRelicApp(config newrelic.Config) newrelic.Application {
	app, err := newrelic.NewApplication(config)

	if err != nil {
		log.Warn("Newrelic agent initialization failed")
		return app
	}

	return app
}
