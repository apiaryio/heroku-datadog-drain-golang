package main

import (
    log "github.com/Sirupsen/logrus"
    "strings"
    "testing"
)

func TestLogProc(t *testing.T) {

    // service , connect histogram value, rest is possibly tags
    //if at==error increment error, rest is possibly tags
    lines := strings.Split(`255 <158>1 2015-04-02T11:52:34.520012+00:00 host heroku router - at=info method=POST path="/users" host=myapp.com request_id=c1806361-2081-42e7-a8aa-92b6808eac8e fwd="24.76.242.18" dyno=web.1 connect=1ms service=37ms status=201 bytes=828
229 <45>1 2015-04-02T11:48:16.839257+00:00 host heroku web.1 - source=web.1 dyno=heroku.35930502.b9de5fce-44b7-4287-99a7-504519070cba sample#load_avg_1m=0.01 sample#load_avg_5m=0.02 sample#load_avg_15m=0.03
222 <134>1 2015-04-07T16:01:43.517062+00:00 host heroku api - Scale to mailer=1, web=3 by someuser@gmail.com`, "\n")

    app := "test"
    tags := []string{"tag1", "tag2"}
    prefix := "prefix"
    s := loadServerCtx()
    s.in = make(chan *logData, 3)
    defer close(s.in)
    s.out = make(chan *logMetrics, 3)
    defer close(s.out)

    go logProcess(s.in, s.out)

    for i, l := range lines {
        log.WithField("line", l).Debug("Sending")
        s.in <- &logData{&app, &tags, &prefix, &lines[i]}
    }

    res := <-s.out
    if res.typ != routerMsg {
        t.Error("result must be ROUTE")
    }

    res = <-s.out
    if res.typ != sampleMsg {
        t.Error("result must be SAMPLE")
    }

    res = <-s.out
    if res.typ != scalingMsg {
        t.Error("result must be SCALE")
    }
}
