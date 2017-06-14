FROM golang:alpine

WORKDIR /go/src/
RUN apk add --no-cache  bash git 


RUN git clone https://github.com/dguihal/goboard app
WORKDIR app
COPY files/start.sh files/


ADD https://github.com/kelseyhightower/confd/releases/download/v0.12.0-alpha3/confd-0.12.0-alpha3-linux-amd64 /usr/local/bin/confd
RUN chmod +x /usr/local/bin/confd

RUN mkdir -p /etc/confd/{conf.d,templates}

COPY files/goboard.yaml.template /etc/confd/templates/
COPY files/confd.toml /etc/confd/conf.d/

RUN go-wrapper download   # "go get -d -v ./..."
RUN go-wrapper install    # "go install -v ./..."

EXPOSE 8080

CMD ["/go/src/app/files/start.sh"]