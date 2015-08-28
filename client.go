package main

import (
	"strconv"
	"strings"

	statsd "github.com/DataDog/datadog-go/statsd"
	log "github.com/Sirupsen/logrus"
)

const sampleRate = 1.0

const (
	routerMsg int = iota
	scalingMsg
	sampleMsg
)

var routerMetricsKeys = []string{"dyno", "method", "status", "path", "host", "code", "desc", "at"}
var sampleMetricsKeys = []string{"source"}
var scalingMetricsKeys = []string{"mailer", "web"}

type Client struct {
	*statsd.Client
}

func statsdClient(addr string) (*Client, error) {

	c, err := statsd.New(addr)
	return &Client{c}, err
}

func (c *Client) sendToStatsd(in chan *logMetrics) {

	var data *logMetrics
	var ok bool
	for {
		data, ok = <-in

		if !ok { //Exit, channel was closed
			return
		}

		log.WithFields(log.Fields{
			"type":   data.typ,
			"app":    data.app,
			"tags":   data.tags,
			"prefix": data.prefix,
		}).Debug("logMetrics received")

		if data.typ == routerMsg {
			c.sendRouterMsg(data)
		} else if data.typ == sampleMsg {
			c.sendSampleMsg(data)
		} else {
			c.sendScalingMsg(data)
		}
	}
}

func (c *Client) sendRouterMsg(data *logMetrics) {

	tags := *data.tags
	for _, mk := range routerMetricsKeys {
		if v, ok := data.metrics[mk]; ok {
			tags = append(tags, mk+":"+v.Val)
		}
	}

	log.WithFields(log.Fields{
		"app":    *data.app,
		"tags":   *data.tags,
		"prefix": *data.prefix,
	}).Debug("sendRouterMsg")

	conn, err := strconv.ParseFloat(data.metrics["connect"].Val, 10)
	if err != nil {
		log.WithFields(log.Fields{
			"type":   "router",
			"err":    err,
			"metric": "connect",
		}).Info("Could not parse metric value")
		return
	}
	serv, err := strconv.ParseFloat(data.metrics["service"].Val, 10)
	if err != nil {
		log.WithFields(log.Fields{
			"type":   "router",
			"metric": "service",
			"err":    err,
		}).Info("Could not parse metric value")
		return
	}

	c.Histogram(*data.prefix+"heroku.router.request.connect", conn, tags, sampleRate)
	c.Histogram(*data.prefix+"heroku.router.request.service", serv, tags, sampleRate)
	if data.metrics["at"].Val == "error" {
		c.Count(*data.prefix+"heroku.router.error", 1, tags, 0.1)
	}
}

func (c *Client) sendSampleMsg(data *logMetrics) {

	tags := *data.tags
	for _, mk := range sampleMetricsKeys {
		if v, ok := data.metrics[mk]; ok {
			tags = append(tags, mk+":"+v.Val)
		}
	}

	log.WithFields(log.Fields{
		"app":    data.app,
		"tags":   data.tags,
		"prefix": data.prefix,
	}).Debug("sendSampleMsg")

	for k, v := range data.metrics {
		if strings.Index(k, "#") != -1 {
			m := strings.Replace(strings.Split(k, "#")[1], "_", ".", -1)
			vnum, err := strconv.ParseFloat(v.Val, 10)
			if err == nil {
				c.Histogram(*data.prefix+"heroku.dyno."+m, vnum, tags, sampleRate)
			} else {
				log.WithFields(log.Fields{
					"type":   "sample",
					"metric": k,
					"err":    err,
				}).Info("Could not parse metric value")
			}
		}
	}
}

func (c *Client) sendScalingMsg(data *logMetrics) {
	tags := *data.tags

	log.WithFields(log.Fields{
		"app":    data.app,
		"tags":   data.tags,
		"prefix": data.prefix,
	}).Debug("sendScalingMsg")

	for _, mk := range scalingMetricsKeys {
		if v, ok := data.metrics[mk]; ok {
			vnum, err := strconv.ParseFloat(v.Val, 10)
			if err == nil {
				c.Gauge(*data.prefix+"heroku.dyno."+mk, vnum, tags, sampleRate)
			} else {
				log.WithFields(log.Fields{
					"type":   "scaling",
					"metric": mk,
					"err":    err,
				}).Info("Could not parse metric value")
			}
		}
	}
}
