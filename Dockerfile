FROM golang:latest
MAINTAINER kn100@kn100.me
LABEL SERVICE mqtt680influxbridge

ENV APP_NAME mqtt680influxbridge

COPY . /go/src/${APP_NAME}
WORKDIR /go/src/${APP_NAME}

RUN go get ./
RUN go build -o ${APP_NAME}
CMD ./${APP_NAME}
