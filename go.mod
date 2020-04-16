module github.com/dguihal/goboard

go 1.14

require (
	github.com/boltdb/bolt v1.3.1
	github.com/dchest/uniuri v0.0.0-20200228104902-7aecb25e1fe5
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.7.4
	github.com/gorilla/websocket v1.4.2
	github.com/hishboy/gocommons v0.0.0-20160108023425-89887b2ade6d
	golang.org/x/crypto v0.0.0-20200406173513-056763e48d71
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	gopkg.in/yaml.v2 v2.2.8
)

replace github.com/dguihal/goboard/cookie => ./cookie

replace github.com/dguihal/goboard/backend => ./backend

replace github.com/dguihal/goboard/user => ./user

replace github.com/dguihal/goboard/utils => ./utils
