package main

import (
	"net"
	"testing"
)

var app = "test"
var tags = []string{"tag1", "tag2"}
var prefix = "prefix."

var statsdTests = []struct {
	cnt      int
	m        logMetrics
	Expected []string
}{
	{
		cnt: 2,
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
				"garbage": {"bar", ""},
			},
		},
		Expected: []string{
			"prefix.heroku.router.request.connect:1.000000|h|#tag1,tag2,at:info",
			"prefix.heroku.router.request.service:37.000000|h|#tag1,tag2,at:info",
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
		},
		Expected: []string{
			"prefix.heroku.dyno.load.avg.1m:0.010000|g|#tag1,tag2,source:web1",
		},
	},
	{
		cnt: 2,
		m: logMetrics{
			scalingMsg,
			&app,
			&tags,
			&prefix,
			map[string]logValue{
				"mailer": {"1", ""},
				"web":    {"3", ""},
			},
		},
		Expected: []string{
			"prefix.heroku.dyno.mailer:1.000000|g|#tag1,tag2",
			"prefix.heroku.dyno.web:3.000000|g|#tag1,tag2",
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

	// c.ExcludedTags = map[string]bool{"path": true}
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
