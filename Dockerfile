FROM golang:1.14-alpine

ENV SWAGGER_PATH="/go/src/goboard/web/swagger" \
    WEBUI_PATH="/go/src/goboard/web/static" \
    GOPATH="/go" \
    GOBOARD_DB_PATH="/var/lib/goboard" \
    GOBOARD_DB_FILE="/var/lib/goboard/goboard.db" \
    GOBOARD_CONFIG_PATH="/etc/goboard" \
    GOBOARD_CONFIG_FILE="/etc/goboard/goboard.yaml" \
    GOBOARD_LOG_PATH="/var/log/goboard"

WORKDIR /${GOPATH}/src/goboard
COPY . .
RUN rm -rf vendor && \
    go get -d -v ./... && \
    go install -v ./... && \
    adduser -S -h "${GOBOARD_DB_PATH}" -D goboard && \
    mkdir -p "${GOBOARD_CONFIG_PATH}" && \
    mkdir -p "${GOBOARD_LOG_PATH}"

COPY dockerfiles/entrypoint.sh /
COPY goboard.yaml "${GOBOARD_CONFIG_FILE}"

RUN chown goboard: "${GOBOARD_CONFIG_PATH}" && \
    chown goboard: "${GOBOARD_LOG_PATH}" && \
    chmod +x /entrypoint.sh

EXPOSE 8080

USER goboard
ENTRYPOINT ["/entrypoint.sh"]
