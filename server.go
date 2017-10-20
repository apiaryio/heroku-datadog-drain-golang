package main

import (
	"bufio"
	"net/http"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent"
)

const bufferLen = 500

type logData struct {
	app    *string
	tags   *[]string
	prefix *string
	line   *string
}

type ServerCtx struct {
	Port        string
	AllowedApps []string
	AppPasswd   map[string]string
	AppTags     map[string][]string
	AppPrefix   map[string]string
	StatsdUrl   string
	Debug       bool
	in          chan *logData
	out         chan *logMetrics
	newRelicApp newrelic.Application
}

//Load configuration from envrionment variables, see list below
//ALLOWED_APPS=my-app,.. Required.
//Comma seperated list of app names

//<APP-NAME>_PASSWORD=.. Required.
//One per allowed app where <APP-NAME> corresponds to an app name from ALLOWED_APPS

//<APP-NAME>_TAGS=mytag,..  Optional.
// Comma seperated list of default tags for each app

//<APP-NAME>_PREFIX=yee     Optional.
//String to be prepended to all metrics from a given app

//STATSD_URL=..  Required. Default: localhost:8125
//DATADOG_DRAIN_DEBUG=         Optional. If DEBUG is set, a lot of stuff will be logged
func loadServerCtx() *ServerCtx {
	newrelicConfig := BuildNewRelicConfig(
		os.Getenv("NEWRELIC_ENABLED"),
		os.Getenv("NEWRELIC_LICENSE_KEY"),
	)

	s := &ServerCtx{
		Port:        "8080",
		AppPasswd:   make(map[string]string),
		AppTags:     make(map[string][]string),
		AppPrefix:   make(map[string]string),
		StatsdUrl:   "localhost:8125",
		Debug:       false,
		newRelicApp: NewNewRelicApp(newrelicConfig),
	}
	port := os.Getenv("PORT")
	if port != "" {
		s.Port = port
	}

	allApps := os.Getenv("ALLOWED_APPS")
	if allApps != "" {
		apps := strings.Split(allApps, ",")
		log.WithField("apps", apps).Info("ALLOWED_APPS loaded.")
		for _, app := range apps {
			name := strings.ToUpper(app)
			s.AllowedApps = append(s.AllowedApps, app)
			s.AppPasswd[app] = os.Getenv(name + "_PASSWORD")
			if s.AppPasswd[app] == "" {
				log.WithField("app", app).Warn("App is allowed but no password set")
			}
			tags := os.Getenv(name + "_TAGS")
			if tags != "" {
				s.AppTags[app] = strings.Split(tags, ",")
			}
			prefix := os.Getenv(name + "_PREFIX")
			if strings.Index(prefix, ".") == -1 {
				s.AppPrefix[app] = prefix + "."
			} else {
				s.AppPrefix[app] = prefix
			}
		}
	} else {
		log.Warn("No Allowed apps set, nobody can access this service!")
	}

	statsd := os.Getenv("STATSD_URL")
	if statsd != "" {
		s.StatsdUrl = statsd
	}

	if os.Getenv("DATADOG_DRAIN_DEBUG") != "" {
		s.Debug = true
	}

	log.WithFields(log.Fields{
		"port":         s.Port,
		"AlloweApps":   s.AllowedApps,
		"AppPasswords": "************",
		"AppTags":      s.AppTags,
		"AppPrefix":    s.AppPrefix,
		"StatsdUrl":    s.StatsdUrl,
		"Debug":        s.Debug,
	}).Info("Configuration loaded")

	return s
}

func init() {
	// Output to stderr instead of stdout
	log.SetOutput(os.Stderr)

	// Only log the Info severity or above.
	log.SetLevel(log.InfoLevel)
}

func (s *ServerCtx) processLogs(c *gin.Context) {
	txn := s.newRelicApp.StartTransaction("/", c.Writer.(http.ResponseWriter), c.Request)
	defer txn.End()

	app := c.MustGet(gin.AuthUserKey).(string)
	tags := s.AppTags[app]
	prefix := s.AppPrefix[app]

	scanner := bufio.NewScanner(c.Request.Body)
	for scanner.Scan() {
		line := scanner.Text()
		log.WithField("line", line).Debug("LINE")
		s.in <- &logData{&app, &tags, &prefix, &line}
	}
	if err := scanner.Err(); err != nil {
		log.Error(err)
	}

	c.String(200, "OK")
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	s := loadServerCtx()
	if s.Debug {
		log.SetLevel(log.DebugLevel)
		gin.SetMode(gin.DebugMode)
	}

	c, err := statsdClient(s.StatsdUrl)
	if err != nil {
		log.WithField("statsdUrl", s.StatsdUrl).Fatal("Could not connect to statsd")
	}

	if v := os.Getenv("EXCLUDED_TAGS"); v != "" {
		for _, t := range strings.Split(v, ",") {
			c.ExcludedTags[t] = true
		}
	}

	r := gin.Default()
	r.GET("/status", func(c *gin.Context) {
		c.String(200, "OK")
	})

	if len(s.AppPasswd) > 0 {
		auth := r.Group("/", gin.BasicAuth(s.AppPasswd))
		auth.POST("/", s.processLogs)
	}

	s.in = make(chan *logData, bufferLen)
	defer close(s.in)
	s.out = make(chan *logMetrics, bufferLen)
	defer close(s.out)
	go logProcess(s.in, s.out)
	go c.sendToStatsd(s.out)
	log.Infoln("Server ready ...")
	r.Run(":" + s.Port)

}
