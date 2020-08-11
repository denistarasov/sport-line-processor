package main

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type linePullerStatus int

const (
	ready linePullerStatus = iota
	notReady
	linesProviderIsUnavailable
)

type linePuller struct {
	sync.Mutex
	linesProviderAddr  string
	sportNames         []string
	storage            *storage
	isLineProviderDown bool
	wg                 *sync.WaitGroup
}

func newLinePuller(ctx context.Context, linesProviderAddr string, sportNames []string, storage *storage, wg *sync.WaitGroup) *linePuller {
	lp := &linePuller{
		Mutex:              sync.Mutex{},
		linesProviderAddr:  linesProviderAddr,
		sportNames:         sportNames,
		storage:            storage,
		isLineProviderDown: false,
		wg:                 wg,
	}

	lp.Lock()
	for _, sportName := range lp.sportNames {
		lp.wg.Add(1)
		go lp.StartLinePullerWorker(ctx, lp.linesProviderAddr, sportName, time.NewTicker(time.Second))
	}
	lp.Unlock()
	return lp
}

func (lp *linePuller) StartLinePullerWorker(ctx context.Context, linesProviderAddr, sportName string, ticker *time.Ticker) {
	log.Infof("starting worker for %s", sportName)
PullingLoop:
	for {
		select {
		case <-ctx.Done():
			break PullingLoop
		case <-ticker.C:
		}
		r, err := http.NewRequest("GET", linesProviderAddr+sportName, nil)
		if err != nil {
			log.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			log.Fatal("could not connect to lines provider: ", err)
		}
		body, _ := ioutil.ReadAll(resp.Body)
		linesMap := map[string]interface{}{}
		err = json.Unmarshal(body, &linesMap)
		if err != nil {
			log.Fatal("lines provider sent data which couldn't be unmarshalled as map: ", err)
		}
		sportMap, ok := linesMap["lines"].(map[string]interface{})
		if !ok {
			log.Fatal(fmt.Sprintf("could not unpack %s map: ", sportName), err)
		}
		sportLine, ok := sportMap[strings.ToUpper(sportName)].(string)
		if !ok {
			log.Fatal("...") // todo
		}
		sportLineDouble, err := strconv.ParseFloat(sportLine, 64)
		if err != nil {
			log.Fatal("cannot convert sportline to double")
		}
		lp.storage.Upload(sportName, sportLineDouble)
		log.Debug(fmt.Sprintf("pulled the line for %s with value %s", sportName, sportLine))
	}
	log.Infof("worker for %s is shut down", sportName)
	lp.wg.Done()
}

func (lp *linePuller) isReady() linePullerStatus {
	lp.Lock()
	defer lp.Unlock()
	if lp.storage.Count() == len(lp.sportNames) {
		return ready
	}
	if lp.isLineProviderDown {
		return linesProviderIsUnavailable
	}
	return notReady
}
