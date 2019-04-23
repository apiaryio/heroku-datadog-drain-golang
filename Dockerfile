FROM golang:1.12 as builder

ENV APP_VERSION 1.2.0

RUN mkdir -p /usr/src/app

COPY . /usr/src/app

RUN cd /usr/src/app && \
    go get ./... && \
    go install

FROM scratch
COPY --from=builder /go/bin/heroku-datadog-drain-golang .
CMD ["./heroku-datadog-drain-golang"]
