package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	viper.SetDefault("grpc_port", 9090)
	viper.SetDefault("db_driver", "sqlite3")
	viper.SetDefault("db_path", "data/echovault.db")
	viper.SetDefault("storage_type", "local")
	viper.SetDefault("storage_path", "data/files")
	viper.AutomaticEnv()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("grpc_port")))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	reflection.Register(s)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutting down server...")
		cancel()
		s.GracefulStop()
	}()

	slog.Info("starting EchoVault server", "port", viper.GetInt("grpc_port"))
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
