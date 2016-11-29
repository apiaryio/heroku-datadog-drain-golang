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
	metricsTag
)

var routerMetricsKeys = []string{"dyno", "method", "status", "path", "host", "code", "desc", "at"}
var sampleMetricsKeys = []string{"source"}
var scalingMetricsKeys = []string{"mailer", "web"}
var customMetricsKeys = []string{"media_type", "output_type", "route"}

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
		} else if data.typ == scalingMsg {
			c.sendScalingMsg(data)
		} else if data.typ == metricsTag {
			c.sendMetricsWithTags(data)
		} else {
			log.WithField("type", data.typ).Warn("Unknown log message")
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

	err = c.Histogram(*data.prefix+"heroku.router.request.connect", conn, tags, sampleRate)
	if err != nil {
		log.WithField("error", err).Info("Failed to send Histogram")
	}
	err = c.Histogram(*data.prefix+"heroku.router.request.service", serv, tags, sampleRate)
	if err != nil {
		log.WithField("error", err).Info("Failed to send Histogram")
	}
	if data.metrics["at"].Val == "error" {
		err = c.Count(*data.prefix+"heroku.router.error", 1, tags, 0.1)
		if err != nil {
			log.WithField("error", err).Info("Failed to send Count")
		}
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
		"app":    *data.app,
		"tags":   tags,
		"prefix": *data.prefix,
	}).Debug("sendSampleMsg")

	for k, v := range data.metrics {
		if strings.Index(k, "#") != -1 {
			m := strings.Replace(strings.Split(k, "#")[1], "_", ".", -1)
			vnum, err := strconv.ParseFloat(v.Val, 10)
			if err == nil {
				err = c.Gauge(*data.prefix+"heroku.dyno."+m, vnum, tags, sampleRate)
				if err != nil {
					log.WithField("error", err).Info("Failed to send Gauge")
				}
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
		"app":    *data.app,
		"tags":   tags,
		"prefix": *data.prefix,
	}).Debug("sendScalingMsg")

	for _, mk := range scalingMetricsKeys {
		if v, ok := data.metrics[mk]; ok {
			vnum, err := strconv.ParseFloat(v.Val, 10)
			if err == nil {
				err = c.Gauge(*data.prefix+"heroku.dyno."+mk, vnum, tags, sampleRate)
				if err != nil {
					log.WithField("error", err).Info("Failed to send Gauge")
				}
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

func (c *Client) sendMetricsWithTags(data *logMetrics) {
	tags := *data.tags

Tags:
	for k, v := range data.metrics {
		if strings.Index(k, "tag#") != -1 {
			if _, err := strconv.Atoi(v.Val); err != nil {
				m := strings.Replace(strings.Split(k, "tag#")[1], "_", ".", -1)
				for _, mk := range customMetricsKeys {
					if m == mk {
						tags = append(tags, mk+":"+v.Val)
						continue Tags
					}
				}
			}
		}
	}

	log.WithFields(log.Fields{
		"app":    *data.app,
		"tags":   tags,
		"prefix": *data.prefix,
	}).Debug("sendMetricTag")

	for k, v := range data.metrics {
		if strings.Index(k, "#") != -1 {
			if vnum, err := strconv.ParseFloat(v.Val, 10); err == nil {
				m := strings.Replace(strings.Split(k, "#")[1], "_", ".", -1)
				err = c.Gauge(*data.prefix+"app.metric."+m, vnum, tags, sampleRate)
				if err != nil {
					log.WithField("error", err).Warning("Failed to send Gauge")
				}
			} else {
				log.WithFields(log.Fields{
					"type":   "metrics",
					"metric": k,
					"err":    err,
				}).Debug("Could not parse metric value")
			}
		}
	}
}
