package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func main() {
	httpAddr := flag.String("http", ":8090", "address for http server")
	grpcAddr := flag.String("grpc", ":8091", "address for grpc server")
	linesProviderAddr := flag.String("provider", "http://localhost:8000/api/v1/lines/", "address for lines provider server")

	sportNameToPullingInterval := make(map[string]int32)
	sportNameToPullingInterval["baseball"] = int32(*flag.Int("baseball", 1, "interval for pulling baseball lines (seconds)"))
	sportNameToPullingInterval["football"] = int32(*flag.Int("football", 1, "interval for pulling football lines (seconds)"))
	sportNameToPullingInterval["soccer"] = int32(*flag.Int("soccer", 1, "interval for pulling soccer lines (seconds)"))

	logLevel := *flag.String("log", "info", "log level, allowed options: debug, info, warn, error, fatal")

	flag.Parse()

	switch strings.ToLower(logLevel) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.Info(fmt.Sprintf("starting program (http_address: %s, grpc_address: %s)", *httpAddr, *grpcAddr))

	storage := NewStorage()

	sportNames := []string{"baseball", "football", "soccer"}
	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	lp := NewLinePuller(ctx, *linesProviderAddr, sportNames, storage, wg)

	// Start HTTP server
	srv := &http.Server{Addr: *httpAddr}
	http.HandleFunc("/ready", readyHandler(lp))
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
		log.Info("server is shut down")
	}()

	// Start gRPC server
	wg.Add(1)
	grpcServer := grpc.NewServer()
	RegisterSportLinesServiceServer(grpcServer, SportLinesPublisherServer{
		storage:                    storage,
		sportNameToPullingInterval: sportNameToPullingInterval,
	})
	go func(s *grpc.Server, serverAddr string) {
		defer wg.Done()
		listener, err := net.Listen("tcp", serverAddr)
		if err != nil {
			log.Fatal(err)
		}
		serverStarted := make(chan struct{})
		err = StartSportLinesPublisher(s, listener, serverStarted)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("grpc server is shut down")
	}(grpcServer, *grpcAddr)

	// Prepare for graceful stop
	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)
	sig := <-shutdownSignals
	log.Infof("received signal (%s), gracefully shutting down...", sig.String())
	cancelFunc()
	err := srv.Shutdown(nil)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.GracefulStop()
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
