FROM golang:1.6

ENV GLIDE_VERSION 0.10.2
ENV APP_VERSION 1.0.2

ADD https://github.com/Masterminds/glide/releases/download/${GLIDE_VERSION}/glide-${GLIDE_VERSION}-linux-amd64.tar.gz /tmp/glide-${GLIDE_VERSION}-linux-amd64.tar.gz
RUN cd /tmp && \
    tar -zxvf /tmp/glide-${GLIDE_VERSION}-linux-amd64.tar.gz && \
    cp /tmp/linux-amd64/glide /usr/local/bin/glide && \
    chmod 755 /usr/local/bin/glide && \
    rm /tmp/glide-${GLIDE_VERSION}-linux-amd64.tar.gz && rm -rf /tmp/linux-amd64/

COPY . /go/src/github.com/apiaryio/heroku-datadog-drain-go

RUN cd /go/src/github.com/apiaryio/heroku-datadog-drain-go && \
    glide install && \
    go install

ENTRYPOINT ["/go/bin/heroku-datadog-drain-go"]
