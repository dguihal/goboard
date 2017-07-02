FROM golang:alpine

WORKDIR /go/src/
RUN apk add --no-cache  bash git

RUN git clone https://github.com/dguihal/goboard

WORKDIR goboard
COPY dockerfiles/start.sh /go/bin/

ENV GOPATH /go
COPY goboard.yaml /go/bin/goboard.yaml.template

RUN go-wrapper download
RUN go-wrapper install

EXPOSE 8080

RUN mv /go/src/goboard/webui /go/bin/webui
RUN mv /go/src/goboard/swagger-ui /go/bin/swaggerui

CMD ["/go/bin/start.sh"]