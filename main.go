package main

import (
	"chat/conf"
	"chat/router"
)

func main() {
	conf.Init()
	r := router.NewRouter()
	r.Run(conf.HttpPort)
}
