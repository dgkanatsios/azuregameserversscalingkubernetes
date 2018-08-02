package main

import (
	"flag"
	"time"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/gc"
	webserver "github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/webserver"
	log "github.com/sirupsen/logrus"
)

func main() {

	rungc := flag.Bool("gc", false, "Run the Garbage Collector. Default: false")
	gcinterval := flag.Int("gcinterval", 1, "Interval in minutes for the Garbage Collector. Default: 1")
	port := flag.Int("port", 8000, "API Server Port. Default: 8000")
	listrunningauth := flag.Bool("listingauth", false, "If true, /running requires authentication. Default: false")

	flag.Parse()

	if *rungc {
		// initialize the garbage collector
		go gc.Run(time.Duration(*gcinterval) * time.Minute)
	}

	err := webserver.Run(*port, *listrunningauth)
	if err != nil {
		log.Fatalf("error creating WebServer: %s", err.Error())
	}
}
