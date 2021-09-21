package server

import (
	"context"
	"fmt"
	"github.com/palantir/stacktrace"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type MinimalGRPCServer struct {
	listenPort uint32
	listenProtocol string
	stopGracePeriod time.Duration  // How long we'll give the server to stop after asking nicely before we kill it
	serviceRegistrationFuncs []func(*grpc.Server)
}

// Creates a minimal gRPC server but doesn't start it
// The service registration funcs will be applied, in order, to register services with the underlying gRPC server object
func NewMinimalGRPCServer(listenPort uint32, listenProtocol string, stopGracePeriod time.Duration, serviceRegistrationFuncs []func(*grpc.Server)) *MinimalGRPCServer {
	return &MinimalGRPCServer{listenPort: listenPort, listenProtocol: listenProtocol, stopGracePeriod: stopGracePeriod, serviceRegistrationFuncs: serviceRegistrationFuncs}
}

func (server MinimalGRPCServer) Run() error {
	loggingInterceptorFunc := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		grpcMethod := info.FullMethod
		logrus.Debugf("Received gRPC request to method '%v' with args:\n%+v", grpcMethod, req)
		resp, err := handler(ctx, req)
		if err != nil {
			logrus.Debugf("gRPC request to method '%v' failed with error:\n%v", grpcMethod, err)
		} else {
			logrus.Debugf("gRPC request to method '%v' succeeded with response:\n%+v", grpcMethod, resp)
		}
		return resp, err
	}
	loggingInterceptor := grpc.UnaryInterceptor(loggingInterceptorFunc)

	grpcServer := grpc.NewServer(loggingInterceptor)

	for _, registrationFunc := range server.serviceRegistrationFuncs {
		registrationFunc(grpcServer)
	}

	listenAddressStr := fmt.Sprintf(":%v", server.listenPort)
	listener, err := net.Listen(server.listenProtocol, listenAddressStr)
	if err != nil {
		return stacktrace.Propagate(
			err,
			"An error occurred creating the listener on %v/%v",
			server.listenProtocol,
			server.listenPort,
		)
	}

	// Signals are used to interrupt the server, so we catch them here
	termSignalChan := make(chan os.Signal, 1)
	signal.Notify(termSignalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	grpcServerResultChan := make(chan error)

	go func() {
		var resultErr error = nil
		if err := grpcServer.Serve(listener); err != nil {
			resultErr = stacktrace.Propagate(err, "The gRPC server exited with an error")
		}
		grpcServerResultChan <- resultErr
	}()

	// Wait until we get a shutdown signal
	<- termSignalChan

	serverStoppedChan := make(chan interface{})
	go func() {
		grpcServer.GracefulStop()
		serverStoppedChan <- nil
	}()
	select {
	case <- serverStoppedChan:
		logrus.Debug("gRPC server has exited gracefully")
	case <- time.After(server.stopGracePeriod):
		logrus.Warnf("gRPC server failed to stop gracefully after %v; hard-stopping now...", server.stopGracePeriod)
		grpcServer.Stop()
		logrus.Debug("gRPC server was forcefully stopped")
	}
	if err := <- grpcServerResultChan; err != nil {
		// Technically this doesn't need to be an error, but we make it so to fail loudly
		return stacktrace.Propagate(err, "gRPC server returned an error after it was done serving")
	}

	return nil
}
