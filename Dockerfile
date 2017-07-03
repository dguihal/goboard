FROM golang:alpine

WORKDIR /go/src/
RUN apk add --no-cache  bash git

RUN git clone -b improve_docker https://github.com/dguihal/goboard

WORKDIR goboard
COPY dockerfiles/start.sh /go/bin/

ENV GOPATH /go
COPY goboard.yaml /go/bin/goboard.yaml.template

RUN go-wrapper download
RUN go-wrapper install

EXPOSE 8080

RUN cp -Rfv /go/src/goboard/webui /go/bin/webui
RUN cp -Rfv /go/src/goboard/swagger-ui /go/bin/swaggerui

CMD ["/go/bin/start.sh"]