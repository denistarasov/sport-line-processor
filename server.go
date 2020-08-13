package main

import (
	"context"
	"io"
	"net"
	"reflect"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCServerError error

var (
	duplicateError   gRPCServerError = status.Error(codes.InvalidArgument, "duplicates in sport list")
	periodicityError gRPCServerError = status.Error(
		codes.InvalidArgument,
		"periodicity of sending lines is more frequent than their pulling periodicity",
	)
	unknownSportNameError gRPCServerError = status.Error(codes.InvalidArgument, "sport name is unknown")
	emptySportListError   gRPCServerError = status.Error(codes.InvalidArgument, "sport list can't be empty")
)

type sportLinesPublisherServer struct {
	storage                    storage
	sportNameToPullingInterval map[string]int32
}

func sender(
	ctx context.Context,
	srv SportLinesService_SubscribeOnSportLinesServer,
	storage storage,
	senderChan <-chan map[string]struct{},
	wg *sync.WaitGroup,
) {
	sportNameToPrevLine := make(map[string]float64)
MainLoop:
	for {
		select {
		case <-ctx.Done():
			break MainLoop
		case update := <-senderChan:
			sportNameToLine := make(map[string]float64)
			if update == nil {
				sportNameToNewLine := make(map[string]float64)
				for sportName, prevSportLine := range sportNameToPrevLine {
					sportLine, _ := storage.Get(sportName)
					sportNameToLine[sportName] = sportLine - prevSportLine
					sportNameToNewLine[sportName] = sportLine
				}
				sportNameToPrevLine = sportNameToNewLine
			} else {
				for sportName := range update {
					sportLine, _ := storage.Get(sportName)
					sportNameToLine[sportName] = sportLine
					sportNameToPrevLine[sportName] = sportLine
				}
			}

			resp := SportLinesResponse{
				SportNameToLine: sportNameToLine,
			}
			err := srv.Send(&resp)
			if err != nil {
				log.Info("error in gRPC Send function: ", err)
			}
		}
	}
	wg.Done()
}

func timer(ctx context.Context, updateChan <-chan update, senderChan chan<- map[string]struct{}, wg *sync.WaitGroup) {
	update := <-updateChan
	ticker := time.NewTicker(time.Second * update.duration)
	senderChan <- update.sportNames
MainLoop:
	for {
		select {
		case <-ctx.Done():
			break MainLoop
		case update = <-updateChan:
			ticker = time.NewTicker(time.Second * update.duration)
			senderChan <- update.sportNames
		case <-ticker.C:
			senderChan <- nil
		}
	}
	wg.Done()
}

type update struct {
	duration   time.Duration
	sportNames map[string]struct{}
}

func (s sportLinesPublisherServer) SubscribeOnSportLines(srv SportLinesService_SubscribeOnSportLinesServer) error {
	log.Info("started gRPC server")

	ctx := srv.Context()

	childCtx, cancelFunc := context.WithCancel(ctx)
	senderChan := make(chan map[string]struct{})
	updateChan := make(chan update)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Add(1)

	go timer(childCtx, updateChan, senderChan, wg)
	go sender(childCtx, srv, s.storage, senderChan, wg)

	prevSports := make(map[string]struct{})
	validSportNames := s.storage.GetKeys()

	for {
		select {
		case <-ctx.Done():
			cancelFunc()
			wg.Wait()

			return ctx.Err()
		default:
		}

		req, err := srv.Recv()
		if err == io.EOF {
			cancelFunc()
			log.Info("connection with client closed due to EOF")

			return nil
		}
		if err != nil {
			log.Errorf("error in gRPC Recv function: %v", err)

			continue
		}

		if len(req.SportNames) == 0 {
			cancelFunc()

			return emptySportListError
		}

		for _, sportName := range req.SportNames {
			_, exists := validSportNames[sportName]
			if !exists {
				cancelFunc()

				return unknownSportNameError
			}

			pullingInterval := s.sportNameToPullingInterval[sportName]
			if pullingInterval > req.TimeInterval {
				cancelFunc()

				return periodicityError
			}
		}

		update := update{
			duration:   time.Duration(req.TimeInterval),
			sportNames: nil,
		}

		curSports := make(map[string]struct{})

		for _, sportName := range req.SportNames {
			_, exists := curSports[sportName]
			if exists {
				cancelFunc()

				return duplicateError
			}

			curSports[sportName] = struct{}{}
		}
		if !reflect.DeepEqual(curSports, prevSports) {
			prevSports = curSports
			update.sportNames = curSports
		}

		updateChan <- update
	}
}

func startSportLinesPublisher(s *grpc.Server, listener net.Listener, serverStarted chan struct{}) error {
	close(serverStarted)

	err := s.Serve(listener)
	if err != nil {
		return err
	}

	return nil
}
