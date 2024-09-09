#!/bin/sh -x
set -e

env

echo "Generating conf"

if [ -n "${MAX_HISTORY_SIZE}" ]; then
	sed -i -e "s#^MaxHistorySize: .*#MaxHistorySize: ${MAX_HISTORY_SIZE}#" "${GOBOARD_CONFIG_FILE}"
fi

if [ -n "${BACKEND_TIME_ZONE}" ]; then
	sed -i -e "s#^BackendTimeZone: .*#BackendTimeZone: ${BACKEND_TIME_ZONE}#" "${GOBOARD_CONFIG_FILE}"
fi

if [ -n "${COOKIE_DURATION}" ]; then
	sed -i -e "s#^CookieDuration: .*#CookieDuration: ${COOKIE_DURATION}#" "${GOBOARD_CONFIG_FILE}"
fi

if [ -z "${ADMIN_TOKEN}" ]; then
	echo "AdminToken MUST be set, aborting"
	exit 1
else
	sed -i -e "s#AdminToken: .*#AdminToken: ${ADMIN_TOKEN}#" "${GOBOARD_CONFIG_FILE}"
fi

sed -i -e "s#^GoBoardDBFile: .*#GoBoardDBFile: ${GOBOARD_DB_FILE}#" "${GOBOARD_CONFIG_FILE}"
sed -i -e "s#^SwaggerPath: .*#SwaggerPath: ${SWAGGER_PATH}#" "${GOBOARD_CONFIG_FILE}"
sed -i -e "s#WebuiPath: .*#WebuiPath: ${WEBUI_PATH}#" "${GOBOARD_CONFIG_FILE}"

if [ -n "${GOBOARD_ACCESSLOG_FILE}" ]; then
	sed -i -e "s#^AccessLogFile: .*#AccessLogFile: ${GOBOARD_ACCESSLOG_FILE}#" "${GOBOARD_CONFIG_FILE}"
else
	sed -i -e "/^AccessLogFile/d" "${GOBOARD_CONFIG_FILE}" # Will write to stdout then
fi

cat "${GOBOARD_CONFIG_FILE}"

echo "Starting Goboard"
ls -l /goboard
/goboard -C "${GOBOARD_CONFIG_FILE}"
