FROM golang:1.6

ENV GO15VENDOREXPERIMENT 1
ENV VERSION 1.0.1

ADD https://github.com/Masterminds/glide/releases/download/0.10.2/glide-0.10.2-linux-amd64.tar.gz /tmp/glide-0.10.2-linux-amd64.tar.gz
RUN cd /tmp && \
    tar -zxvf /tmp/glide-0.10.2-linux-amd64.tar.gz && \
    cp /tmp/linux-amd64/glide /usr/local/bin/glide && \
    chmod 755 /usr/local/bin/glide && \
    rm /tmp/glide-0.10.2-linux-amd64.tar.gz && rm -rf /tmp/linux-amd64/

COPY . /go/src/github.com/apiaryio/heroku-datadog-drain-go

RUN cd /go/src/github.com/apiaryio/heroku-datadog-drain-go && \
    glide install && \
    go install

ENTRYPOINT ["/go/bin/heroku-datadog-drain-go"]
