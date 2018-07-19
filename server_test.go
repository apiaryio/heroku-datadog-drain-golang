package main

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

var fullTests = []struct {
	cnt      int
	Req      string
	Expected []string
}{
	{
		cnt: 2,
		Req: `255 <158>1 2015-04-02T11:52:34.520012+00:00 host heroku router - at=info method=POST path="/users" host=myapp.com request_id=c1806361-2081-42e7-a8aa-92b6808eac8e fwd="24.76.242.18" dyno=web.1 connect=1ms service=37ms status=201 bytes=828`,
		Expected: []string{
			"heroku.router.request.connect:1.000000|h|#at:info,dyno:web.1,host:myapp.com,method:POST,path:/users,status:201,statusFamily:2xx",
			"heroku.router.request.service:37.000000|h|#at:info,dyno:web.1,host:myapp.com,method:POST,path:/users,status:201,statusFamily:2xx",
		},
	},
	{
		cnt: 1,
		Req: `229 <45>1 2015-04-02T11:48:16.839257+00:00 host heroku web.1 - source=web.1 dyno=heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba sample#load_avg_1m=0.01`,
		Expected: []string{
			"heroku.dyno.load.avg.1m:0.010000|g|#source:web.1,type:web",
		},
	},
	{
		cnt: 3,
		Req: `222 <134>1 2015-04-07T16:01:43.517062+00:00 host app api - Scaled to web@3:Performance-L mailer@1:Standard-2X by user someuser@gmail.com`,
		Expected: []string{
			"_e{16,77}:heroku/api: test|Scaled to web@3:Performance-L mailer@1:Standard-2X by user someuser@gmail.com",
			"heroku.dyno.mailer:1.000000|g",
			"heroku.dyno.web:3.000000|g",
		},
	},
	{
		cnt: 1,
		Req: `222 <134>1 2015-04-07T16:01:43.517062+00:00 host app api - Release v1 created by foo@bar`,
		Expected: []string{
			"_e{13,29}:app/api: test|Release v1 created by foo@bar",
		},
	},
	{
		cnt: 9,
		Req: `452 <134>1 2015-04-07T16:01:43.517062+00:00 host app web.1 - info: responseLogger: metric#tag#route=/parser metric#request_id=11747467-f4ce-4b06-8c99-92be968a02e3 metric#request_length=541 metric#response_length=5163 metric#parser_time=5ms metric#eventLoop.count=606 metric#eventLoop.avg_ms=515.503300330033 metric#eventLoop.p50_ms=0.8805309734513275 metric#eventLoop.p95_ms=3457.206896551724 metric#eventLoop.p99_ms=3457.206896551724 metric#eventLoop.max_ms=5008`,
		Expected: []string{
			"app.metric.request.length:541.000000|g|#route:/parser,source:web.1,type:web",
			"app.metric.response.length:5163.000000|g|#route:/parser,source:web.1,type:web",
			"app.metric.parser.time:5.000000|g|#route:/parser,source:web.1,type:web",
			"app.metric.eventLoop.count:606.000000|g|#route:/parser,source:web.1,type:web",
			"app.metric.eventLoop.avg.ms:515.503300|g|#route:/parser,source:web.1,type:web",
			"app.metric.eventLoop.p50.ms:0.880531|g|#route:/parser,source:web.1,type:web",
			"app.metric.eventLoop.p95.ms:3457.206897|g|#route:/parser,source:web.1,type:web",
			"app.metric.eventLoop.p99.ms:3457.206897|g|#route:/parser,source:web.1,type:web",
			"app.metric.eventLoop.max.ms:5008.000000|g|#route:/parser,source:web.1,type:web",
		},
	},
	{
		cnt: 1,
		Req: `127 <134>1 2015-04-07T16:01:43.517062+00:00 host app web.1 - info: responseLogger: metric#tag#custom=test metric#request_length=541`,
		Expected: []string{
			"app.metric.request.length:541.000000|g|#custom:test,source:web.1,type:web",
		},
	},
	{
		cnt: 1,
		Req: `152 <134>1 2015-04-07T16:01:43.517062+00:00 host app web.1 - info: responseLogger: metric#tag#custom=test metric#tag#custom2=test2 metric#request_length=541`,
		Expected: []string{
			"app.metric.request.length:541.000000|g|#custom2:test2,custom:test,source:web.1,type:web",
		},
	},
}

func TestStatusRequest(t *testing.T) {

	r := gin.New()
	r.GET("/status", func(c *gin.Context) {
		c.String(200, "OK")
	})

	req, _ := http.NewRequest("GET", "/status", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}

	if string(body) != "OK" {
		t.Error("resp body should match")
	}

	if resp.Code != 200 {
		t.Error("should get a 200")
	}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func TestLogRequest(t *testing.T) {

	s := loadServerCtx()
	s.AllowedApps = append(s.AllowedApps, "test")
	s.AppPasswd["test"] = "pass"

	s.in = make(chan *logData)
	defer close(s.in)
	s.out = make(chan *logMetrics)
	defer close(s.out)

	go logProcess(s.in, s.out)

	r := gin.New()
	auth := r.Group("/", gin.BasicAuth(s.AppPasswd))
	auth.POST("/", s.processLogs)

	req, _ := http.NewRequest("POST", "/", bytes.NewBuffer([]byte("LINE of text\nAnother line\n")))
	req.SetBasicAuth("test", "pass")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	if string(body) != "OK" {
		t.Error("resp body should match")
	}

	if resp.Code != 200 {
		t.Error("should get a 200")
	}

}

func TestFull(t *testing.T) {

	s := loadServerCtx()
	s.AllowedApps = append(s.AllowedApps, "test")
	s.AppPasswd["test"] = "pass"

	s.in = make(chan *logData)
	defer close(s.in)
	s.out = make(chan *logMetrics)
	defer close(s.out)

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

	go logProcess(s.in, s.out)
	go c.sendToStatsd(s.out)

	r := gin.New()
	auth := r.Group("/", gin.BasicAuth(s.AppPasswd))
	auth.POST("/", s.processLogs)

	data := make([]byte, 1024)
	for _, tt := range fullTests {
		req, _ := http.NewRequest("POST", "/", bytes.NewBuffer([]byte(tt.Req)))
		req.SetBasicAuth("test", "pass")
		resp := httptest.NewRecorder()
		r.ServeHTTP(resp, req)

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Error(err)
		}
		if string(body) != "OK" {
			t.Error("resp body should match")
		}
		if resp.Code != 200 {
			t.Error("should get a 200")
		}
		if tt.cnt != len(tt.Expected) {
			t.Error("Count of expected results isn't equal to inputs")
		}
		for i := 0; i < tt.cnt; i++ {
			n, err := server.Read(data)
			if err != nil {
				t.Fatal(err)
			}
			message := data[:n]
			findEqual := false

			for j := 0; j < len(tt.Expected); j++ {
				if string(message) == tt.Expected[j] {
					findEqual = true
				}
			}
			if findEqual == false {
				t.Errorf("Expected: %s. Actual: %s", tt.Expected[i], string(message))
			}
		}
	}

	time.Sleep(1 * time.Second)
}
