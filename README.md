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

Support user launched docker (-u parameter)

Support for BasePath, MaxHistorySize, BackendTimeZone, CookieDuration and AdminToken environment variables (-e)

**AdminToken has to be set (no default)**

Usage:

`git clone https://github.com/dguihal/goboard`

`cd goboard`

`docker build -t goboard:latest .`

Create a data volume (To store db data)
`docker volume create goboard_data`

`docker run -p 8080:8080 -u 1001 goboard_data:/data -e AdminToken=somekindofverylongstring -e DBDataPath=/data goboard`
