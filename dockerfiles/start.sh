#!/bin/bash
set -e

cd /go/bin/

echo "Generating conf"
cp goboard.yaml.template goboard.yaml

if [ ! -z "${BasePath}" ] ; then
    sed -i -e "s/BasePath: .*/BasePath: ${BasePath}/" goboard.yaml
fi

if [ ! -z "${MaxHistorySize}" ] ; then
    sed -i -e "s/MaxHistorySize.*/MaxHistorySize: ${MaxHistorySize}/" goboard.yaml
fi

if [ ! -z "${BackendTimeZone}" ] ; then
    sed -i -e "s/BackendTimeZone.*/BackendTimeZone: ${BackendTimeZone}/" goboard.yaml
fi

if [ ! -z "${CookieDuration}" ] ; then
    sed -i -e "s/CookieDuration.*/CookieDuration: ${CookieDuration}/" goboard.yaml
fi

if [ -z "${GoBoardDBPath}" ] ; then
    echo "Warn: GoBoardDBFile won't be stored on a persitent file system"
else
    sed -i -e "s#GoBoardDBFile.*#GoBoardDBFile: ${GoBoardDBPath}/goboard.db#" goboard.yaml
fi

if [ -z "${AdminToken}" ] ; then
    echo "AdminToken MUST be set, aborting"
    test $AdminToken
else
    sed -i -e "s/AdminToken.*/AdminToken: ${AdminToken}/" goboard.yaml    
fi

sed -i -e "s/SwaggerPath.*/SwaggerPath: swaggerui/" goboard.yaml
sed -i -e "s/WebuiPath.*/WebuiPath: webui/" goboard.yaml

USER="goboard"
adduser -D -u 1000 goboard
chown -R "${USER}" "${GoBoardDBPath}"

echo "Starting Goboard"
su-exec "${USER}" goboard

