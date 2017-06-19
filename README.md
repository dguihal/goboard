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

** AdminToken has to be set (no default)

Usage:

`git clone https://github.com/dguihal/goboard`

`cd goboard`

`docker build -t goboard:latest .`

`docker run -p 8080:8080 -u 1001 -e AdminToken=somekindofverylongstring goboard`
