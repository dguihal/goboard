
##
## Build
##
FROM golang:1.14-alpine AS build

WORKDIR /goboard

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./
COPY internal ./internal/

RUN go build -o /goboard

##
## Run
##
FROM alpine:latest

ENV SWAGGER_PATH="/var/lib/goboard/web/swagger" \
    WEBUI_PATH="/var/lib/goboard/web/static" \
    GOBOARD_DB_PATH="/var/lib/goboard" \
    GOBOARD_DB_FILE="/var/lib/goboard/goboard.db" \
    GOBOARD_CONFIG_PATH="/etc/goboard" \
    GOBOARD_CONFIG_FILE="/etc/goboard/goboard.yaml" \
    GOBOARD_LOG_PATH="/var/log/goboard"

WORKDIR /

RUN apk add --no-cache tzdata && \
    adduser -S -h "${GOBOARD_DB_PATH}" -D goboard -u 1000 && \
    mkdir -p "${GOBOARD_CONFIG_PATH}" && \
    mkdir -p "${GOBOARD_LOG_PATH}"

COPY --from=build /goboard/goboard /
COPY dockerfiles/entrypoint.sh /
COPY goboard.yaml "${GOBOARD_CONFIG_FILE}"
COPY dockerfiles/entrypoint.sh /
COPY goboard.yaml "${GOBOARD_CONFIG_FILE}"
COPY web/swagger/ "${SWAGGER_PATH}"
COPY api/swagger.yaml "${SWAGGER_PATH}"
COPY web/static/ "${WEBUI_PATH}"

RUN chown goboard: "${GOBOARD_CONFIG_PATH}" && \
    chown goboard: "${GOBOARD_LOG_PATH}" && \
    chmod +x /entrypoint.sh && \
    chmod +x /goboard

EXPOSE 8080

USER goboard
ENTRYPOINT ["/entrypoint.sh"]
