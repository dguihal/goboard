FROM golang:alpine

WORKDIR /go/src/
RUN apk add --no-cache  bash git

RUN git clone https://github.com/dguihal/goboard
RUN git clone https://github.com/swagger-api/swagger-ui/
RUN 

WORKDIR goboard
COPY files/start.sh /go/bin/

ENV GOPATH /go
#RUN cp files/goboard.yaml.template /etc/confd/templates/
#RUN cp files/confd.toml /etc/confd/conf.d/
COPY goboard.yaml /go/bin/goboard.yaml.template

RUN go-wrapper download
RUN go-wrapper install

EXPOSE 8080

RUN mv /go/src/goboard/webui /go/bin/
RUN mv /go/src/swagger-ui/dist /go/bin/swagger-ui
CMD ["/go/bin/start.sh"]