FROM golang:alpine

WORKDIR /go/src/
RUN apk add --no-cache  bash git

RUN git clone https://github.com/dguihal/goboard
RUN git clone https://github.com/swagger-api/swagger-ui/


WORKDIR goboard
COPY files/start.sh /go/bin/

ENV GOPATH /go
COPY goboard.yaml /go/bin/goboard.yaml.template

RUN go-wrapper download
RUN go-wrapper install

EXPOSE 8080

RUN mv /go/src/goboard/webui /go/bin/
RUN mv /go/src/swagger-ui/dist /go/bin/swaggerui
RUN cp /go/src/goboard/swagger.yaml /go/bin/swaggerui/
RUN sed -i -e "s#http://petstore.swagger.io/v2/swagger.json#swagger.yaml#" /go/bin/swaggerui/index.html

CMD ["/go/bin/start.sh"]