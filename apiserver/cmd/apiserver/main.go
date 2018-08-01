package main

import (
	"flag"
	"time"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/gc"
	webserver "github.com/dgkanatsios/azuregameserversscalingkubernetes/apiserver/webserver"
	log "github.com/sirupsen/logrus"
)

func main() {

	rungc := flag.Bool("gc", false, "Run the Garbage Collector")
	gcinterval := flag.Int("gcinterval", 1, "Interval in minutes for the Garbage Collector")
	port := flag.Int("port", 8000, "API Server Port")

	flag.Parse()

	if *rungc {
		// initialize the garbage collector
		go gc.Run(time.Duration(*gcinterval) * time.Minute)
	}

	err := webserver.Run(*port)
	if err != nil {
		log.Fatalf("error creating WebServer: %s", err.Error())
	}
}
