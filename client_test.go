package main

import (
	"net"
	"testing"
)

var app = "test"
var tags = []string{"tag1", "tag2"}
var prefix = "prefix."
var events = []string{""}

var statsdTests = []struct {
	cnt      int
	m        logMetrics
	Expected []string
}{
	{
		cnt: 3,
		m: logMetrics{
			routerMsg,
			&app,
			&tags,
			&prefix,
			map[string]logValue{
				"at":      {"info", ""},
				"path":    {"/foo", ""},
				"connect": {"1", "ms"},
				"service": {"37", "ms"},
				"status":  {"401", ""},
				"bytes":   {"244000", ""},
				"garbage": {"bar", ""},
			},
			events,
		},
		Expected: []string{
			"prefix.heroku.router.response.bytes:244000.000000|h|#at:info,status:401,tag1,tag2,statusFamily:4xx",
			"prefix.heroku.router.request.connect:1.000000|h|#at:info,status:401,tag1,tag2,statusFamily:4xx",
			"prefix.heroku.router.request.service:37.000000|h|#at:info,status:401,tag1,tag2,statusFamily:4xx",
		},
	},
	{
		cnt: 1,
		m: logMetrics{
			metricsTag,
			&app,
			&tags,
			&prefix,
			map[string]logValue{
				"metric#load_avg_2m": {"0.01", ""},
			},
			events,
		},
		Expected: []string{
			"prefix.app.metric.load.avg.2m:0.010000|g|#tag1,tag2",
		},
	},
	{
		cnt: 1,
		m: logMetrics{
			metricsTag,
			&app,
			&tags,
			&prefix,
			map[string]logValue{
				"sample#load_avg_1m": {"0.01", ""},
			},
			events,
		},
		Expected: []string{
			"prefix.app.metric.load.avg.1m:0.010000|g|#tag1,tag2",
		},
	},
	{
		cnt: 1,
		m: logMetrics{
			metricsTag,
			&app,
			&tags,
			&prefix,
			map[string]logValue{
				"count#clicks": {"1", ""},
			},
			events,
		},
		Expected: []string{
			"prefix.app.metric.clicks:1|c|#tag1,tag2",
		},
	},
	{
		cnt: 1,
		m: logMetrics{
			metricsTag,
			&app,
			&tags,
			&prefix,
			map[string]logValue{
				"measure#temperature": {"1.3", ""},
			},
			events,
		},
		Expected: []string{
			"prefix.app.metric.temperature:1.300000|h|#tag1,tag2",
		},
	},
	{
		cnt: 1,
		m: logMetrics{
			sampleMsg,
			&app,
			&tags,
			&prefix,
			map[string]logValue{
				"source":             {"web1", ""},
				"sample#load_avg_1m": {"0.01", ""},
			},
			events,
		},
		Expected: []string{
			"prefix.heroku.dyno.load.avg.1m:0.010000|g|#source:web1,tag1,tag2",
		},
	},
	{
		cnt: 3,
		m: logMetrics{
			scalingMsg,
			&app,
			&tags,
			&prefix,
			map[string]logValue{
				"mailer": {"1", ""},
				"web":    {"3", ""},
			},
			[]string{
				"Scaling dynos mailer=1 web=3 by foo@bar",
			},
		},
		Expected: []string{
			"_e{16,39}:heroku/api: test|Scaling dynos mailer=1 web=3 by foo@bar|#tag1,tag2",
			"prefix.heroku.dyno.mailer:1.000000|g|#tag1,tag2",
			"prefix.heroku.dyno.web:3.000000|g|#tag1,tag2",
		},
	},
	{
		cnt: 1,
		m: logMetrics{
			releaseMsg,
			&app,
			&tags,
			&prefix,
			map[string]logValue{},
			[]string{
				"Release v1 created by foo@bar",
			},
		},
		Expected: []string{
			"_e{13,29}:app/api: test|Release v1 created by foo@bar|#tag1,tag2",
		},
	},
}

func TestStatsdClient(t *testing.T) {

	addr := "localhost:1201"
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}

	server, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	c, err := statsdClient(addr)
	if err != nil {
		t.Fatal(err)
	}

	c.ExcludedTags["path"] = true

	out := make(chan *logMetrics)
	defer close(out)
	go c.sendToStatsd(out)

	bytes := make([]byte, 1024)
	for _, tt := range statsdTests {
		out <- &tt.m
		for i := 0; i < tt.cnt; i++ {
			n, err := server.Read(bytes)
			if err != nil {
				t.Fatal(err)
			}
			message := bytes[:n]
			if string(message) != tt.Expected[i] {
				t.Errorf("Expected: %s. Actual: %s", tt.Expected[i], string(message))
			}
		}
	}
}
