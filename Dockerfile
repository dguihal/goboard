FROM golang:alpine

RUN apk add --no-cache bash git su-exec

WORKDIR /go/src/

RUN git clone -b improve_docker https://github.com/dguihal/goboard

WORKDIR goboard
COPY dockerfiles/start.sh /go/bin/

ENV GOPATH /go
COPY goboard.yaml /go/bin/goboard.yaml.template

RUN go-wrapper download
RUN go-wrapper install

RUN cp -Rfv /go/src/goboard/webui /go/bin/webui
RUN cp -Rfv /go/src/goboard/swagger-ui /go/bin/swaggerui

EXPOSE 8080

ENTRYPOINT ["/go/bin/start.sh"]