package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"math"
	"net"
	"sync"
	"testing"
	"time"
)

var eps = 0.0001

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(log.WarnLevel)
}

func initServer(t *testing.T, storage *storage, sportNameToPullingInterval map[string]int32) string {
	serverAddr := "localhost:0"
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	serverStarted := make(chan struct{})
	s := grpc.NewServer()
	RegisterSportLinesServiceServer(s, sportLinesPublisherServer{
		storage:                    storage,
		sportNameToPullingInterval: sportNameToPullingInterval,
	})

	go func(s *grpc.Server, listener net.Listener, serverStarted chan struct{}) {
		err := startSportLinesPublisher(s, listener, serverStarted)
		if err != nil {
			t.Error("server finished with error: ", err)
		}
	}(s, listener, serverStarted)

	<-serverStarted

	serverAddr = fmt.Sprintf("localhost:%d", listener.Addr().(*net.TCPAddr).Port)
	return serverAddr
}

func initClient(t *testing.T, serverAddr string) SportLinesService_SubscribeOnSportLinesClient {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		t.Fatal("can't dial to server, err:", err)
	}

	client := NewSportLinesServiceClient(conn)
	stream, err := client.SubscribeOnSportLines(context.Background())
	if err != nil {
		t.Fatal("client was unable to start the stream, err:", err)
	}
	return stream
}

func TestGRPCServer_Ping(t *testing.T) {
	storage := newStorage()
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	req := &SportLinesRequest{
		SportNames:   []string{"soccer"},
		TimeInterval: 1,
	}
	err := stream.Send(req)
	require.NoError(t, err)
}

func TestGRPCServer_Simple(t *testing.T) {
	storage := newStorage()
	sportName := "soccer"
	sportLine := 0.5
	storage.Upload(sportName, sportLine)
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	req := &SportLinesRequest{
		SportNames:   []string{sportName},
		TimeInterval: 1,
	}
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}

	require.Equal(t, map[string]float64{sportName: sportLine}, resp.SportNameToLine)
}

func TestGRPCServer_Interval(t *testing.T) {
	storage := newStorage()
	sportName := "soccer"
	sportLine := 0.5
	storage.Upload(sportName, sportLine)
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	timeInterval := int32(2)
	req := &SportLinesRequest{
		SportNames:   []string{sportName},
		TimeInterval: timeInterval,
	}
	start := time.Now()
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	require.Equal(t, map[string]float64{sportName: sportLine}, resp.SportNameToLine)

	resp, err = stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	require.Equal(t, 1, len(resp.SportNameToLine))
	elapsedTime := int32(time.Since(start).Seconds())
	require.GreaterOrEqual(t, elapsedTime, timeInterval)
}

func TestGRPCServer_Deltas(t *testing.T) {
	storage := newStorage()
	sportName := "soccer"
	sportLine := 0.5
	storage.Upload(sportName, sportLine)
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	req := &SportLinesRequest{
		SportNames:   []string{sportName},
		TimeInterval: 1,
	}
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}

	_, err = stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}

	delta := 0.1
	storage.Upload(sportName, sportLine+delta)
	resp, err := stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	require.Equal(t, 1, len(resp.SportNameToLine))
	require.LessOrEqual(t, math.Abs(delta-resp.SportNameToLine[sportName]), eps)
}

func TestGRPCServer_ManySports(t *testing.T) {
	storage := newStorage()
	sportName := "soccer"
	sportLine := 0.5
	sportName2 := "baseball"
	sportLine2 := 0.6
	storage.Upload(sportName, sportLine)
	storage.Upload(sportName2, sportLine2)
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	req := &SportLinesRequest{
		SportNames:   []string{sportName, sportName2},
		TimeInterval: 1,
	}
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	resp, err := stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	require.Equal(t, map[string]float64{sportName: sportLine, sportName2: sportLine2}, resp.SportNameToLine)
}

func TestGRPCServer_IntervalChange(t *testing.T) {
	storage := newStorage()
	sportName := "soccer"
	sportLine := 0.5
	storage.Upload(sportName, sportLine)
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	timeInterval := int32(2)
	req := &SportLinesRequest{
		SportNames:   []string{sportName},
		TimeInterval: timeInterval,
	}
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	start := time.Now()
	_, err = stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	_, err = stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	elapsedTime := int32(time.Since(start).Seconds())
	require.GreaterOrEqual(t, elapsedTime, timeInterval)

	timeInterval--
	req = &SportLinesRequest{
		SportNames:   []string{sportName},
		TimeInterval: timeInterval,
	}
	start = time.Now()
	err = stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	_, err = stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	_, err = stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	elapsedTime = int32(time.Since(start).Seconds())
	require.GreaterOrEqual(t, elapsedTime, timeInterval)
}

func TestGRPCServer_NoDeltasAfterSportChanges(t *testing.T) {
	storage := newStorage()
	sportName := "soccer"
	sportLine := 0.5
	sportName2 := "baseball"
	sportLine2 := 0.6
	storage.Upload(sportName, sportLine)
	storage.Upload(sportName2, sportLine2)
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	req := &SportLinesRequest{
		SportNames:   []string{sportName},
		TimeInterval: 1,
	}
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	resp, err := stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	require.Equal(t, map[string]float64{sportName: sportLine}, resp.SportNameToLine)

	req = &SportLinesRequest{
		SportNames:   []string{sportName, sportName2},
		TimeInterval: 1,
	}
	err = stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	resp, err = stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	require.Equal(t, map[string]float64{sportName: sportLine, sportName2: sportLine2}, resp.SportNameToLine)
}

func TestGRPCServer_ManySubscribers(t *testing.T) {
	storage := newStorage()
	sportName := "soccer"
	sportLine := 0.5
	sportName2 := "baseball"
	sportLine2 := 0.6
	storage.Upload(sportName, sportLine)
	storage.Upload(sportName2, sportLine2)
	serverAddr := initServer(t, storage, nil)

	clientFunc := func(serverAddr, sportName string, sportLine float64, timeInterval int32, wg *sync.WaitGroup, errors chan error) {
		stream := initClient(t, serverAddr)
		req := &SportLinesRequest{
			SportNames:   []string{sportName},
			TimeInterval: timeInterval,
		}
		err := stream.Send(req)
		if err != nil {
			errors <- fmt.Errorf("client was unable to send request, err: %s", err)
		}

		start := time.Now()

		resp, err := stream.Recv()
		if err != nil {
			errors <- fmt.Errorf("client was unable to receive response, err: %s", err)
		}
		require.Equal(t, map[string]float64{sportName: sportLine}, resp.SportNameToLine)

		resp, err = stream.Recv()
		if err != nil {
			errors <- fmt.Errorf("client was unable to receive response, err: %s", err)
		}

		elapsedTime := int32(time.Since(start).Seconds())
		require.GreaterOrEqual(t, elapsedTime, timeInterval)
		require.Equal(t, map[string]float64{sportName: 0}, resp.SportNameToLine)
		wg.Done()
		errors <- nil
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	errors := make(chan error)
	go clientFunc(serverAddr, sportName, sportLine, 1, wg, errors)
	go clientFunc(serverAddr, sportName2, sportLine2, 2, wg, errors)
	for i := 0; i != 2; i++ {
		err := <-errors
		if err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
}

func TestGRPCServer_DeltasIfSportListDidntChange(t *testing.T) {
	// send 2 times same list of sports
	// todo
	storage := newStorage()
	sportName := "soccer"
	sportLine := 0.5
	storage.Upload(sportName, sportLine)
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	timeInterval := int32(1)
	req := &SportLinesRequest{
		SportNames:   []string{sportName},
		TimeInterval: timeInterval,
	}
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	_, err = stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}

	delta := 0.25
	storage.Upload(sportName, sportLine+delta)
	req = &SportLinesRequest{
		SportNames:   []string{sportName},
		TimeInterval: timeInterval + 1,
	}
	err = stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	resp, err := stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	require.Equal(t, 1, len(resp.SportNameToLine))
	require.LessOrEqual(t, math.Abs(delta-resp.SportNameToLine[sportName]), eps)
}

func TestGRPCServer_GetInvalidSportName(t *testing.T) {
	storage := newStorage()
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	req := &SportLinesRequest{
		SportNames:   []string{"tennis"},
		TimeInterval: 1,
	}
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	_, err = stream.Recv()
	require.Error(t, err)
	require.Equal(t, unknownSportNameError.Error(), err.Error())
}

func TestGRPCServer_EmptySportNamesList(t *testing.T) {
	storage := newStorage()
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	req := &SportLinesRequest{
		SportNames:   []string{},
		TimeInterval: 1,
	}
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	_, err = stream.Recv()
	require.Error(t, err)
	require.Equal(t, emptySportListError.Error(), err.Error())
}

func TestGRPCServer_SportNamesDuplicates(t *testing.T) {
	storage := newStorage()
	storage.Upload("football", 0.1)
	storage.Upload("soccer", 0.2)
	serverAddr := initServer(t, storage, nil)
	stream := initClient(t, serverAddr)

	req := &SportLinesRequest{
		SportNames:   []string{"football", "soccer", "football"},
		TimeInterval: 1,
	}
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	_, err = stream.Recv()
	require.Error(t, err)
	require.Equal(t, duplicateError.Error(), err.Error())
}

func TestGRPCServer_IntervalLessThanStorageUpdate(t *testing.T) {
	// todo other tests for GRPC errors
	storage := newStorage()
	storage.Upload("football", 0.1)
	serverAddr := initServer(t, storage, map[string]int32{"football": 2})
	stream := initClient(t, serverAddr)

	req := &SportLinesRequest{
		SportNames:   []string{"football"},
		TimeInterval: 1,
	}
	err := stream.Send(req)
	if err != nil {
		t.Fatal("client was unable to send request, err:", err)
	}
	_, err = stream.Recv()
	require.Error(t, err)
	require.Equal(t, periodicityError.Error(), err.Error())
}
