package main

import (
	"flag"

	webserver "github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/webserver"
	log "github.com/sirupsen/logrus"
)

func main() {

	port := flag.Int("port", 8000, "API Server Port. Default: 8000")
	listrunningauth := flag.Bool("listingauth", false, "If true, /running requires authentication. Default: false")

	flag.Parse()

	err := webserver.Run(*port, *listrunningauth)
	if err != nil {
		log.Fatalf("error creating WebServer: %s", err.Error())
	}
}
