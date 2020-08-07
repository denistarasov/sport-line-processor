package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"net"
	"reflect"
	"time"
)

type SportLinesPublisherServer struct {
	storage *Storage
}

func sender(ctx context.Context, srv SportLinesService_SubscribeOnSportLinesServer, storage *Storage, senderChan <-chan map[string]struct{}) {
	sportNameToPrevLine := make(map[string]float64)
	for {
		select {
		case <-ctx.Done():
			break
		case update := <-senderChan:
			sportNameToLine := make(map[string]float64)
			if update == nil {
				sportNameToNewLine := make(map[string]float64)
				for sportName, prevSportLine := range sportNameToPrevLine {
					sportLine, exists := storage.Get(sportName)
					if !exists {
						// todo
						log.Fatal("key doesn't exist")
					}
					sportNameToLine[sportName] = sportLine - prevSportLine
					sportNameToNewLine[sportName] = sportLine
				}
				sportNameToPrevLine = sportNameToNewLine
			} else {
				for sportName := range update {
					sportLine, exists := storage.Get(sportName)
					if !exists {
						// todo
					}
					sportNameToLine[sportName] = sportLine
					sportNameToPrevLine[sportName] = sportLine
				}
			}

			resp := SportLinesResponse{
				SportNameToLine: sportNameToLine,
			}
			err := srv.Send(&resp)
			if err != nil {
				log.Info("error in GRPC Send function: ", err)
			}
		}
	}
}

func timer(ctx context.Context, updateChan chan Update, senderChan chan<- map[string]struct{}) {
	update := <-updateChan
	ticker := time.NewTicker(update.duration)
	senderChan <- update.sportNames
	for {
		select {
		case <-ctx.Done():
			break
		case update = <-updateChan:
			ticker = time.NewTicker(update.duration)
			senderChan <- update.sportNames
		case <-ticker.C:
			senderChan <- nil
		}
	}
}

type Update struct {
	duration time.Duration
	areSportNamesUpdated bool
	sportNames map[string]struct{}
}

func (s SportLinesPublisherServer) SubscribeOnSportLines(srv SportLinesService_SubscribeOnSportLinesServer) error {
	log.Info("started GRPC server")
	ctx := srv.Context()
	timerCtx, _ := context.WithCancel(ctx) // todo cancelFunc
	senderChan := make(chan map[string]struct{})
	updateChan := make(chan Update)
	go timer(timerCtx, updateChan, senderChan)
	go sender(ctx, srv, s.storage, senderChan)

	//var prevDuration *int32 = nil
	prevSports := make(map[string]struct{})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := srv.Recv()
		//req.SportNames
		// todo if sport names are the same, only change ticker time

		//close(s.done)
		//s.done = make(chan struct{})

		if err == io.EOF {
			log.Info("connection with client closed")
			return nil
		}
		if err != nil {
			log.Errorf("error in GRPC Recv function: %v", err)
			continue
		}

		update := Update{
			duration:             time.Duration(req.TimeInterval),
			areSportNamesUpdated: false,
			sportNames:           nil,
		}

		//if prevDuration == nil || *prevDuration != req.TimeInterval {
		//	*prevDuration = req.TimeInterval
		//	durations <- time.Duration(req.TimeInterval)
		//}

		curSports := make(map[string]struct{})
		for _, sportName := range req.SportNames {
			curSports[sportName] = struct{}{}
		}
		if !reflect.DeepEqual(curSports, prevSports) {
			prevSports = curSports
			update.areSportNamesUpdated = true
			update.sportNames = curSports
			// todo
		}

		updateChan <- update
	}
}

func StartSportLinesPublisher(storage *Storage, listener net.Listener, serverStarted chan struct{}) error {
	s := grpc.NewServer()
	RegisterSportLinesServiceServer(s, SportLinesPublisherServer{
		storage: storage,
	})
	close(serverStarted)
	err := s.Serve(listener)
	if err != nil {
		//log.Fatal("Failed to serve: ", err)
		return err
	}
	return nil
}

//func main() {
//	StartSportLinesPublisher(":8787")
//}
