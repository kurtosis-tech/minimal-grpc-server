package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/kurtosis-tech/stacktrace"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	// gRPC servers can ONLY run on TCP
	// See https://stackoverflow.com/questions/65111895/using-udp-in-grpc
	listenProtocol = "tcp"
)

type MinimalGRPCServer struct {
	listenPort               uint16
	stopGracePeriod          time.Duration // How long we'll give the server to stop after asking nicely before we kill it
	serviceRegistrationFuncs []func(*grpc.Server)

	// serverCert is optional. If present, server will run HTTPS
	serverCert *tls.Certificate
	// certPool is optional and should be present only if serverCert is present.
	// If present, server will run two-way SSL
	certPool *x509.CertPool
}

// NewMinimalGRPCServer Creates a minimal gRPC server but doesn't start it
// The service registration funcs will be applied, in order, to register services with the underlying gRPC server object
func NewMinimalGRPCServer(listenPort uint16, stopGracePeriod time.Duration, serviceRegistrationFuncs []func(*grpc.Server)) *MinimalGRPCServer {
	return &MinimalGRPCServer{
		listenPort:               listenPort,
		stopGracePeriod:          stopGracePeriod,
		serviceRegistrationFuncs: serviceRegistrationFuncs,
		certPool:                 nil,
		serverCert:               nil,
	}
}

// NewMinimalHttpsGRPCServer is similar to NewMinimalGRPCServer but accepts SSL CA and certificate to enable HTTPS
func NewMinimalHttpsGRPCServer(listenPort uint16, stopGracePeriod time.Duration, certPool *x509.CertPool, serverCert *tls.Certificate, serviceRegistrationFuncs []func(*grpc.Server)) *MinimalGRPCServer {
	return &MinimalGRPCServer{
		listenPort:               listenPort,
		stopGracePeriod:          stopGracePeriod,
		serviceRegistrationFuncs: serviceRegistrationFuncs,
		certPool:                 certPool,
		serverCert:               serverCert,
	}
}

// RunUntilInterrupted runs the server synchronously until an interrupt signal is received
func (server MinimalGRPCServer) RunUntilInterrupted() error {
	// Signals are used to interrupt the server, so we catch them here
	termSignalChan := make(chan os.Signal, 1)
	signal.Notify(termSignalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	serverStopChan := make(chan struct{}, 1)
	go func() {
		<-termSignalChan
		interruptSignal := struct{}{}
		serverStopChan <- interruptSignal
	}()
	if err := server.RunUntilStopped(serverStopChan); err != nil {
		return stacktrace.Propagate(err, "An error occurred running the server using the interrupt channel for stopping")
	}
	return nil
}

// RunUntilStopped runs the server synchronously until a signal is received on the given channel
func (server MinimalGRPCServer) RunUntilStopped(stopper <-chan struct{}) error {
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
	serverOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(loggingInterceptorFunc),
	}

	// Add TLS credentials to server options if certificate have been provided
	tlsCredentialsMaybe := server.loadTlsCredentials()
	if tlsCredentialsMaybe != nil {
		serverOptions = append(serverOptions, grpc.Creds(tlsCredentialsMaybe))
	}
	grpcServer := grpc.NewServer(serverOptions...)
	for _, registrationFunc := range server.serviceRegistrationFuncs {
		registrationFunc(grpcServer)
	}

	listenAddressStr := fmt.Sprintf(":%v", server.listenPort)
	listener, err := net.Listen(listenProtocol, listenAddressStr)
	if err != nil {
		return stacktrace.Propagate(
			err,
			"An error occurred creating the listener on %v/%v",
			listenProtocol,
			server.listenPort,
		)
	}

	grpcServerResultChan := make(chan error)
	mux := cmux.New(listener)
	grpcWebL := mux.Match(cmux.HTTP1Fast())
	grpcL := mux.Match(cmux.Any())

	go func() {
		if resultErr := grpcServer.Serve(grpcL); resultErr != nil {
			logrus.Debugf("error ocurred while creating grpc server: %v", resultErr)
			grpcServerResultChan <- resultErr
		}
	}()

	grpcWebServer := grpcweb.WrapServer(grpcServer, grpcweb.WithOriginFunc(func(origin string) bool { return true }))
	httpServer := &http.Server{
		Handler: http.Handler(grpcWebServer),
	}

	go func() {
		if resultErr := httpServer.Serve(grpcWebL); resultErr != nil {
			logrus.Debugf("error ocurred while creating grpcweb server: %v", resultErr)
			grpcServerResultChan <- resultErr
		}
	}()

	go func() {
		if resultErr := mux.Serve(); resultErr != nil {
			logrus.Debugf("error ocurred while creating mux server: %v", resultErr)
			grpcServerResultChan <- resultErr
		}
	}()

	// Wait until we get a shutdown signal
	<-stopper

	serverStoppedChan := make(chan interface{})
	go func() {
		grpcServer.Stop()
		mux.Close()
		serverStoppedChan <- nil
	}()
	select {
	case <-serverStoppedChan:
		logrus.Debug("gRPC server has exited gracefully")
	case <-time.After(server.stopGracePeriod):
		logrus.Warnf("gRPC server failed to stop gracefully after %v; hard-stopping now...", server.stopGracePeriod)
		grpcServer.Stop()
		mux.Close()
		logrus.Debug("gRPC server was forcefully stopped")
	}

	if err := <-grpcServerResultChan; err != nil {
		// Technically this doesn't need to be an error, but we make it so to fail loudly
		// this is expected behaviour observed via cmux
		gracefulExit := isGracefulExit(err)
		if !gracefulExit {
			return stacktrace.Propagate(err, "gRPC server returned an error after it was done serving")
		}
	}

	return nil
}
func (server MinimalGRPCServer) loadTlsCredentials() credentials.TransportCredentials {
	if server.serverCert == nil {
		// No certificate provided, will use HTTP
		return nil
	}
	if server.certPool == nil {
		// with no cert pool, 1 way SSL will be enabled, no need for CA
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{
				*server.serverCert,
			},
			ClientAuth: tls.NoClientCert,
		}
		return credentials.NewTLS(tlsConfig)
	}
	// 2 ways SSL enabled - Load CA and set ClientAuth to "Require And Verify"
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			*server.serverCert,
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  server.certPool,
	}
	return credentials.NewTLS(tlsConfig)
}

func isGracefulExit(err error) bool {
	switch err {
	case nil, http.ErrServerClosed, cmux.ErrListenerClosed, cmux.ErrServerClosed:
		// do nothing, normal exit
		return true
	default:
		if strings.Contains(err.Error(), "use of closed network connection") {
			return true
		}
		return false
	}
}
