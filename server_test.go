package main

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"math"
	"net"
	"sync"
	"testing"
	"time"
	log "github.com/sirupsen/logrus"
)

var eps = 0.0001

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetLevel(log.WarnLevel)
}

func initServer(t *testing.T, storage *Storage) string {
	serverAddr := "localhost:0"
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		t.Fatal(err)
	}
	serverStarted := make(chan struct{})
	go func(listener net.Listener) {
		err := StartSportLinesPublisher(storage, listener, serverStarted)
		if err != nil {
			t.Fatal("server finished with error: ", err)
		}
		fmt.Println("server finished")
	}(listener)

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
	storage := NewStorage()
	serverAddr := initServer(t, storage)
	stream := initClient(t, serverAddr)

	req := &SportLinesRequest{
		SportNames:   []string{}, // todo list cannot be empty
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

	require.Equal(t, map[string]float64(nil), resp.SportNameToLine)
}

func TestGRPCServer_Simple(t *testing.T) {
	storage := NewStorage()
	sportName := "soccer"
	sportLine := 0.5
	storage.Upload(sportName, sportLine)
	serverAddr := initServer(t, storage)
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
	storage := NewStorage()
	sportName := "soccer"
	sportLine := 0.5
	storage.Upload(sportName, sportLine)
	serverAddr := initServer(t, storage)
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
	storage := NewStorage()
	sportName := "soccer"
	sportLine := 0.5
	storage.Upload(sportName, sportLine)
	serverAddr := initServer(t, storage)
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

	delta := 0.1
	storage.Upload(sportName, sportLine + delta)
	resp, err = stream.Recv()
	if err != nil {
		t.Fatal("client was unable to receive response, err:", err)
	}
	require.Equal(t, 1, len(resp.SportNameToLine))
	require.LessOrEqual(t, math.Abs(delta - resp.SportNameToLine[sportName]), eps)
}

func TestGRPCServer_ManySports(t *testing.T) {
	storage := NewStorage()
	sportName := "soccer"
	sportLine := 0.5
	sportName2 := "baseball"
	sportLine2 := 0.6
	storage.Upload(sportName, sportLine)
	storage.Upload(sportName2, sportLine2)
	serverAddr := initServer(t, storage)
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
	storage := NewStorage()
	sportName := "soccer"
	sportLine := 0.5
	storage.Upload(sportName, sportLine)
	serverAddr := initServer(t, storage)
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

	timeInterval -= 1
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
	storage := NewStorage()
	sportName := "soccer"
	sportLine := 0.5
	sportName2 := "baseball"
	sportLine2 := 0.6
	storage.Upload(sportName, sportLine)
	storage.Upload(sportName2, sportLine2)
	serverAddr := initServer(t, storage)
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
	storage := NewStorage()
	sportName := "soccer"
	sportLine := 0.5
	sportName2 := "baseball"
	sportLine2 := 0.6
	storage.Upload(sportName, sportLine)
	storage.Upload(sportName2, sportLine2)
	serverAddr := initServer(t, storage)

	clientFunc := func(serverAddr, sportName string, sportLine float64, timeInterval int32, wg *sync.WaitGroup) {
		stream := initClient(t, serverAddr)
		req := &SportLinesRequest{
			SportNames:   []string{sportName},
			TimeInterval: timeInterval,
		}
		err := stream.Send(req)
		if err != nil {
			t.Fatal("client was unable to send request, err:", err)
		}

		start := time.Now()

		resp, err := stream.Recv()
		if err != nil {
			t.Fatal("client was unable to receive response, err:", err)
		}
		require.Equal(t, map[string]float64{sportName: sportLine}, resp.SportNameToLine)

		resp, err = stream.Recv()
		if err != nil {
			t.Fatal("client was unable to receive response, err:", err)
		}

		elapsedTime := int32(time.Since(start).Seconds())
		require.GreaterOrEqual(t, elapsedTime, timeInterval)
		require.Equal(t, map[string]float64{sportName: 0}, resp.SportNameToLine)
		wg.Done()
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go clientFunc(serverAddr, sportName, sportLine, 1, wg)
	go clientFunc(serverAddr, sportName2, sportLine2, 2, wg)
	wg.Wait()
}

func TestGRPCServer_DeltasIfSportListDidntChange(t *testing.T) {
	// send 2 times same list of sports
	// todo
	storage := NewStorage()
	sportName := "soccer"
	sportLine := 0.5
	storage.Upload(sportName, sportLine)
	serverAddr := initServer(t, storage)
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
	storage.Upload(sportName, sportLine + delta)
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
	require.LessOrEqual(t, math.Abs(delta - resp.SportNameToLine[sportName]), eps)
}

// todo other tests for GRPC errors
func TestGRPCServer_GetInvalidSportName(t *testing.T) {}
func TestGRPCServer_IntervalLessThanStorageUpdate(t *testing.T) {}
func TestGRPCServer_SportNamesDuplicates(t *testing.T) {}
func TestGRPCServer_EmptySportNamesList(t *testing.T) {}
