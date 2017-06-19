#!/bin/bash
set -e
set -x

cd /go/bin/

echo "Using user"
id

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

if [ -z "${AdminToken}" ] ; then
    echo "AdminToken MUST be set, aborting"
    test $AdminToken
else
    sed -i -e "s/AdminToken.*/AdminToken: ${AdminToken}/" goboard.yaml    
fi

sed -i -e "s/SwaggerPath.*/SwaggerPath: swaggerui/" goboard.yaml
sed -i -e "s/WebuiPath.*/WebuiPath: webui/" goboard.yaml

echo "Starting Goboard"
exec goboard

