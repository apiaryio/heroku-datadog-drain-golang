[![Build Status](https://travis-ci.org/apiaryio/heroku-datadog-drain-golang.svg?branch=master)](https://travis-ci.org/apiaryio/heroku-datadog-drain-golang)

# Heroku Datadog Drain

Golang version of [NodeJS](https://github.com/ozinc/heroku-datadog-drain)

Funnel metrics from multiple Heroku apps into DataDog using statsd.

## Supported Heroku metrics:

- Heroku Router response times, status codes, etc.
- Application errors
- Custom metrics
- Heroku Dyno [runtime metrics](https://devcenter.heroku.com/articles/log-runtime-metrics)

## Get Started

### Clone the Github repository

```bash
git clone git@github.com:apiaryio/heroku-datadog-drain-golang.git
cd heroku-datadog-drain-golang
```

### Setup Heroku, specify the app(s) you'll be monitoring and create a password for each.

```
heroku create
heroku config:set ALLOWED_APPS=<your-app-slug> <YOUR-APP-SLUG>_PASSWORD=<password>
```

> **OPTIONAL**: Setup Heroku build packs, including the Datadog DogStatsD client.
If you already have a StatsD client running, see the STATSD_URL configuration option below.


```
heroku buildpacks:add heroku/go
heroku buildpacks:add --index 1 https://github.com/miketheman/heroku-buildpack-datadog.git
heroku config:set HEROKU_APP_NAME=$(heroku apps:info|grep ===|cut -d' ' -f2)
heroku config:add DATADOG_API_KEY=<your-Datadog-API-key>
```

### Deploy to Heroku.

```
git push heroku master
heroku ps:scale web=1
```

### Add the Heroku log drain using the app slug and password created above.

```
heroku drains:add https://<your-app-slug>:<password>@<this-log-drain-app-slug>.herokuapp.com/ --app <your-app-slug>
```

## Configuration
```bash
STATSD_URL=..             # Required. Set to: localhost:8125
DATADOG_API_KEY=...       # Required. Datadog API Key - https://app.datadoghq.com/account/settings#api
ALLOWED_APPS=my-app,..    # Required. Comma seperated list of app names
<APP-NAME>_PASSWORD=..    # Required. One per allowed app where <APP-NAME> corresponds to an app name from ALLOWED_APPS
<APP-NAME>_TAGS=mytag,..  # Optional. Comma seperated list of default tags for each app
<APP-NAME>_PREFIX=..      # Optional. String to be prepended to all metrics from a given app
DATADOG_DRAIN_DEBUG=..    # Optional. If DEBUG is set, a lot of stuff will be logged :)
EXCLUDED_TAGS: path,host  # Optional. Recommended to solve problem with tags limit (1000)
```
Note that the capitalized `<APP-NAME>` and `<YOUR-APP-SLUG>` appearing above indicate that your application name and slug should also be in full caps. For example, to set the password for an application named `my-app`, you would need to specify `heroku config:set ALLOWED_APPS=my-app MY-APP_PASSWORD=example_password`

The rationale for `EXCLUDED_TAGS` is that the `path=` tag in Heroku logs includes the full HTTP path - including, for instance, query parameters. This makes is very easy to swarm Datadog with numerous distinct tag/value pairs; and Datadog has a hard limit of 1000 such distinct pairs. When the limit is breached, they blacklist the entire metric.

## Heroku settings

You need use Standard dynos and better and enable `log-runtime-metrics` in heroku labs for every application.

```bash
heroku labs:enable log-runtime-metrics -a APP_NAME
```

This adds basic metrics (cpu, memory etc.) into logs.

## Custom Metrics

If you want to log some custom metrics just format the log line like following:

```
app web.1 - info: responseLogger: metric#tag#route=/parser metric#request_id=11747467-f4ce-4b06-8c99-92be968a02e3 metric#request_length=541 metric#response_length=5163 metric#parser_time=5ms metric#eventLoop.count=606 metric#eventLoop.avg_ms=515.503300330033 metric#eventLoop.p50_ms=0.8805309734513275 metric#eventLoop.p95_ms=3457.206896551724 metric#eventLoop.p99_ms=3457.206896551724 metric#eventLoop.max_ms=5008
```
We support `metric#` for values and `metric#tag` for tags.
