package main

import (
	"github.com/DataDog/datadog-go/statsd"
	"log"
)

func sendMessage() {
	c, err := statsd.New("127.0.0.1:8125")
	if err != nil {
		log.Fatal(err)
	}
	// prefix every metric with the app name
	c.Namespace = "flubber."
	// send the EC2 availability zone as a tag with every metric
	c.Tags = append(c.Tags, "us-east-1a")
	err = c.Gauge("request.duration", 1.2, nil, 1)
	// ...
}
