# goboard

## Introduction

## How-to

### Build

`export GOPATH=~/go`

`go get github.com/dguihal/goboard`

`cd $GOPATH/src/github.com/dguihal/goboard`

`go build`

### Configure

### Run

### Dockerfile

Support for BasePath, MaxHistorySize, BackendTimeZone, CookieDuration and AdminToken environment variables (-e)

Runs as uid 1000

**AdminToken has to be set (no default)**

Usage:

`git clone https://github.com/dguihal/goboard`

`cd goboard`

`docker build -t goboard:latest .`

#### With data out of docker instance (Let you upgrade easily without losing your data)

Create a data volume (To store db data)
`docker volume create goboard_data`

Run (Beware of your AdminToken : It's the key to protect your admin rights)
`docker run -p 8080:8080 -v goboard_data:/data -e AdminToken=somekindofverylongstring -e GoBoardDBPath=/data goboard`

#### With data inside the docker instance (Beware : Destroying your image destroys data)

`docker run -p 8080:8080 -e AdminToken=somekindofverylongstring goboard`
