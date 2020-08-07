package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	// http address
	// grpc address
	// baseball, football, soccer intervals
	// logging level
	httpAddress := flag.String("http", ":0", "desired address for http server")
	grpcAddress := flag.String("grpc", ":0", "desired address for grpc server")
	//linesProviderAddress := flag.String("provider", "", "address for lines provider server")
	flag.Int("baseball", 1, "interval for pulling baseball lines (seconds)")
	flag.Int("football", 1, "interval for pulling football lines (seconds)")
	flag.Int("soccer", 1, "interval for pulling soccer lines (seconds)")
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(log.DebugLevel)
	log.Info(fmt.Sprintf("starting program (http_address: %s, grpc_address: %s)", *httpAddress, *grpcAddress))

	storage := NewStorage()

	sportNames := []string{"baseball", "football", "soccer"}
	linesProviderAddr := "http://localhost:8000/api/v1/lines/"
	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	lp := NewLinePuller(ctx, linesProviderAddr, sportNames, storage, wg)

	srv := &http.Server{Addr: ":8090"}
	http.HandleFunc("/ready", readyHandler(lp))
	wg.Add(1)
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
		log.Info("server is shut down")
		wg.Done()
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)

	sig := <-shutdownSignals
	log.Infof("received signal (%s), gracefully shutting down...", sig.String())
	cancelFunc()
	err := srv.Shutdown(nil)
	if err != nil {
		log.Fatal(err)
	}
	wg.Wait()
}

func readyHandler(lp *LinePuller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("/ready: received request")
		encoder := json.NewEncoder(w)
		status := lp.isReady()
		switch status {
		case Ready:
			log.Info("/ready: status ok")
			w.WriteHeader(http.StatusOK)
			encoder.Encode(map[string]string{
				"response": "OK",
			})
		case NotReady:
			w.WriteHeader(http.StatusServiceUnavailable)
			log.Info("/ready: not all sports were pulled yet")
			encoder.Encode(map[string]string{
				"response": "Please try later",
			})
		case LinesProviderIsUnavailable:
			log.Info("/ready: lines provider is not available at all")
			w.WriteHeader(http.StatusServiceUnavailable)
			encoder.Encode(map[string]string{
				"response": "Service is unavailable",
			})
		}
	}
}
