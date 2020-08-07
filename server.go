package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"io"
	"net"
	"reflect"
	"sync"
	"time"
)

type SportLinesPublisherServer struct {
	storage *Storage
}

func sender(ctx context.Context, srv SportLinesService_SubscribeOnSportLinesServer, storage *Storage, senderChan <-chan map[string]struct{}, wg *sync.WaitGroup) {
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
	wg.Done()
}

func timer(ctx context.Context, updateChan <-chan Update, senderChan chan<- map[string]struct{}, wg *sync.WaitGroup) {
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

type Update struct {
	duration             time.Duration
	sportNames           map[string]struct{}
}

func (s SportLinesPublisherServer) SubscribeOnSportLines(srv SportLinesService_SubscribeOnSportLinesServer) error {
	log.Info("started GRPC server")
	ctx := srv.Context()
	childCtx, _ := context.WithCancel(ctx) // todo cancelFunc
	senderChan := make(chan map[string]struct{})
	updateChan := make(chan Update)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go timer(childCtx, updateChan, senderChan, wg)
	go sender(childCtx, srv, s.storage, senderChan, wg)

	prevSports := make(map[string]struct{})

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		default:
		}

		req, err := srv.Recv()
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
			sportNames:           nil,
		}

		curSports := make(map[string]struct{})
		for _, sportName := range req.SportNames {
			curSports[sportName] = struct{}{}
		}
		if !reflect.DeepEqual(curSports, prevSports) {
			prevSports = curSports
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
