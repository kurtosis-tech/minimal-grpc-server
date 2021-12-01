package server

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
	"time"
)

const (
	listenPort = uint16(9003)
	stopGracePeriod = 10 * time.Second
)

func TestManualShutdown(t *testing.T) {
	server := NewMinimalGRPCServer(listenPort, stopGracePeriod, []func(*grpc.Server){})

	stopChan := make(chan interface{})
	runResultChan := make(chan error)
	go func() {
		runResultChan <- server.RunUntilStopped(stopChan)
	}()
	select {
	case err := <- runResultChan:
		t.Fatalf("The server unexpectedly returned a result before we sent a signal to stop it:\n%v", err)
	case <- time.After(1 * time.Second):
		// Nothing
	}
	stopChan <- "Stop now"

	maxWaitForServerToStop := 1 * time.Second
	var errAfterStop error
	select {
	case errAfterStop = <- runResultChan:
		// Assignment is all we have to do
	case <- time.After(maxWaitForServerToStop):
		t.Fatalf("Expected the server to have stopped after %v, but it didn't", maxWaitForServerToStop)
	}
	assert.NoError(t, errAfterStop, "Expected the server to exit without an error but it threw the following:\n%v", errAfterStop)
}
