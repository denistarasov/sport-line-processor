package main

import (
	"context"
	"encoding/json"
	"flag"
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

	baseballInterval := flag.Int("baseball", 1, "interval for pulling baseball lines (seconds)")
	footballInterval := flag.Int("football", 1, "interval for pulling football lines (seconds)")
	soccerInterval := flag.Int("soccer", 1, "interval for pulling soccer lines (seconds)")

	logLevel := flag.String("log", "info", "log level, allowed options: debug, info, warn, error, fatal")

	flag.Parse()

	sportNameToPullingInterval := make(map[string]int32)
	sportNameToPullingInterval["baseball"] = int32(*baseballInterval)
	sportNameToPullingInterval["football"] = int32(*footballInterval)
	sportNameToPullingInterval["soccer"] = int32(*soccerInterval)

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	switch strings.ToLower(*logLevel) {
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
	default:
		log.Fatalf("unknown log level: %s", *logLevel)
	}

	log.Infof("starting program (http_address: %s, grpc_address: %s, provider address: %s)", *httpAddr, *grpcAddr, *linesProviderAddr)

	storage := newDBStorage()

	sportNames := []string{"baseball", "football", "soccer"}
	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	lp := newLinePuller(ctx, *linesProviderAddr, sportNames, storage, wg)

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
	RegisterSportLinesServiceServer(grpcServer, sportLinesPublisherServer{
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
		err = startSportLinesPublisher(s, listener, serverStarted)
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
	err := srv.Shutdown(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.GracefulStop()
	wg.Wait()
}

func readyHandler(lp *linePuller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Info("/ready: received request")
		encoder := json.NewEncoder(w)
		status := lp.isReady()
		switch status {
		case ready:
			log.Info("/ready: status ok")
			w.WriteHeader(http.StatusOK)
			_ = encoder.Encode(map[string]string{
				"response": "OK",
			})
		case notReady:
			w.WriteHeader(http.StatusServiceUnavailable)
			log.Info("/ready: not all sports were pulled yet")
			_ = encoder.Encode(map[string]string{
				"response": "Please try later",
			})
		case linesProviderIsUnavailable:
			log.Info("/ready: lines provider is not available at all")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = encoder.Encode(map[string]string{
				"response": "Service is unavailable",
			})
		}
	}
}
