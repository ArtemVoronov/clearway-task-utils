package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	ConfigFilePathEnvVar                     = "CONFIG_FILE_PATH"
	DefaultConfigFilePath                    = ".env"
	DefaultHttpServerCertificateFilePath     = "server.crt"
	DefaultHttpServerKeyFilePath             = "server.key"
	DefaultHttpServerReadTimeout             = 15 * time.Minute
	DefaultHttpServerWriteTimeout            = 15 * time.Minute
	DefaultHttpServerGracefulShutdownTimeout = 2 * time.Minute
)

type FuncSetup func()

type FuncShutdown func()

type ServerConfig struct {
	ReadTimeout             time.Duration
	WriteTimeout            time.Duration
	GracefulShutdownTimeout time.Duration
	Host                    string
	Routes                  *http.ServeMux
	CertificateFilePath     string
	KeyFilePath             string
	Setup                   FuncSetup
	Shutdown                FuncShutdown
}

func Start(config *ServerConfig) {
	if config.Setup != nil {
		config.Setup()
	}
	if config.Shutdown != nil {
		defer config.Shutdown()
	}

	server := &http.Server{
		Handler:      config.Routes,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		Addr:         config.Host,
	}

	idleCloseConnections := make(chan struct{})
	go func() {
		interruptSignals := make(chan os.Signal, 1)
		signal.Notify(interruptSignals, syscall.SIGINT, syscall.SIGTERM)
		<-interruptSignals
		log.Println("server shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("server forced to shutdown: %v", err)
		}
		close(idleCloseConnections)
	}()

	if err := server.ListenAndServeTLS(config.CertificateFilePath, config.KeyFilePath); err != http.ErrServerClosed {
		log.Fatal(err)
	}

	<-idleCloseConnections

	log.Println("server has been shutdown")
}
