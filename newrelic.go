package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/newrelic/go-agent"
	"net/http"
	"os"
)

type NewRelicClient struct {
	app newrelic.Application
}

func (nrc *NewRelicClient) StartTransaction(name string,
	w http.ResponseWriter,
	r *http.Request) newrelic.Transaction {
	if nrc.app == nil {
		return nil
	}

	return nrc.app.StartTransaction(name, w, r)
}

func (nrc *NewRelicClient) EndTransaction(txn newrelic.Transaction) {
	if txn != nil {
		txn.End()
	}
}

func BuildNewRelicConfig() newrelic.Config {
	disabled := os.Getenv("NEWRELIC_ENABLED") == "false" || os.Getenv("NEWRELIC_LICENSE_KEY") == ""

	config := newrelic.NewConfig("Roo Datadog Bridge", os.Getenv("NEWRELIC_LICENSE_KEY"))
	if disabled {
		config.Enabled = false
	}

	return config
}

func NewNewRelicClient() *NewRelicClient {
	client := &NewRelicClient{}

	app, err := newrelic.NewApplication(BuildNewRelicConfig())

	if err != nil {
		log.Warn("Newrelic agent initialization failed")
		return client
	}

	client.app = app
	return client
}
