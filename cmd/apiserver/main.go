package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apiserver/apiserver"
	"github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apiserver/webhookserver"

	log "github.com/sirupsen/logrus"
)

func main() {

	port := flag.Int("port", 8000, "API Server Port. Default: 8000")
	webhookport := flag.Int("whport", 8001, "WebHook Server Port. Default: 8001")
	listrunningauth := flag.Bool("listingauth", false, "If true, /running requires authentication. Default: false")

	flag.Parse()

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	apiserver := apiserver.Run(*port, *listrunningauth)
	webhookserver := webhookserver.Run("/certificate/cert.pem", "/certificate/key.pem", *webhookport)

	<-signalChan

	log.Infof("Got OS shutdown signal, shutting down webhook and API servers gracefully...")
	apiserver.Shutdown(context.Background())
	webhookserver.Shutdown(context.Background())
}
