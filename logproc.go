package main

import (
	"bytes"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/kr/logfmt"
)

type logValue struct {
	Val  string
	Unit string // (e.g. ms, MB, etc)
}

type logMetrics struct {
	typ     int
	app     *string
	tags    *[]string
	prefix  *string
	metrics map[string]logValue
	events  []string
}

var dynoNumber *regexp.Regexp = regexp.MustCompile(`\.\d+$`)

func (lm *logMetrics) HandleLogfmt(key, val []byte) error {

	i := bytes.LastIndexFunc(val, isDigit)
	if i == -1 {
		lm.metrics[string(key)] = logValue{string(val), ""}
	} else {
		lm.metrics[string(key)] = logValue{string(val[:i+1]), string(val[i+1:])}
	}

	log.WithFields(log.Fields{
		"key":  string(key),
		"val":  lm.metrics[string(key)].Val,
		"unit": lm.metrics[string(key)].Unit,
	}).Debug("logMetric")

	return nil
}

// return true if r is an ASCII digit only, as opposed to unicode.IsDigit.
func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func parseMetrics(typ int, ld *logData, data *string, out chan *logMetrics) {
	var myslice []string
	lm := logMetrics{typ, ld.app, ld.tags, ld.prefix, make(map[string]logValue, 5), myslice}
;
	if typ == releaseMsg {
		events := append(lm.events, *data)
		lm.events = events
		out <- &lm
		return
	}

	if err := logfmt.Unmarshal([]byte(*data), &lm); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Warn()
		return
	}
	if source, ok := lm.metrics["source"]; ok {
		tags := append(*lm.tags, "type:"+dynoNumber.ReplaceAllString(source.Val, ""))
		lm.tags = &tags
	}
	out <- &lm
}

var scalingRe = regexp.MustCompile("Scaled to (.*) by user .*")
var scaledDynoRe = regexp.MustCompile("([^@ ]*)@([^: ]*):([^ ]*)")

func parseScalingMessage(ld *logData, message *string, out chan *logMetrics) {
	if scalingInfo := scalingRe.FindStringSubmatch(*message); scalingInfo != nil {
		scaledDynoInfos := scaledDynoRe.FindAllStringSubmatch(scalingInfo[1], -1)
		logValues := make(map[string]logValue)
		for _, dynoInfo := range scaledDynoInfos {
			dynoName := dynoInfo[1]
			count := dynoInfo[2]
			dynoType := dynoInfo[3]
			log.WithFields(log.Fields{
				"dynoName": dynoName,
				"count": count,
				"dynoType": dynoType,
			}).Debug()
			logValues[dynoName] = logValue{count, dynoType}
		}
		events := []string{*message}
		lm := logMetrics{scalingMsg, ld.app, ld.tags, ld.prefix, logValues, events}
		out <- &lm
	} else {
		log.WithFields(log.Fields{
			"err": "Scaling message not matched",
			"message": *message,
		}).Warn()
	}
}

func logProcess(in chan *logData, out chan *logMetrics) {

	var data *logData
	var ok bool
	for {
		data, ok = <-in

		if !ok { //Exit, channel was closed
			return
		}

		log.Debugln(*data.line)
		var output []string
		output = strings.Split(*data.line, " - ")
		if len(output) < 2 {
			output = strings.Split(*data.line, " app[heroku-redis]: ")
		}
		if len(output) < 2 {
			continue
		}
		headers := strings.Split(strings.TrimSpace(output[0]), " ")
		if len(headers) >= 6 {
			headers = headers[3:6]
		}

		log.WithField("headers", headers).Debug("Line headers")
		if strings.Contains(output[1], "source=REDIS") {
			parseMetrics(sampleMsg, data, &output[1], out)
		} else if headers[1] == "heroku" {
			if headers[2] == "router" {
				parseMetrics(routerMsg, data, &output[1], out)
			} else {
				parseMetrics(sampleMsg, data, &output[1], out)
			}
		} else if headers[1] == "app" {
			if headers[2] == "api" {
				if strings.HasPrefix(output[1], "Release") {
					parseMetrics(releaseMsg, data, &output[1], out)
				} else {
					parseScalingMessage(data, &output[1], out)
				}
			} else {
				dynoType := dynoNumber.ReplaceAllString(headers[2], "")
				tags := append(*data.tags, "source:"+headers[2], "type:"+dynoType)
				data.tags = &tags
				parseMetrics(metricsTag, data, &output[1], out)
			}
		}
	}
}
