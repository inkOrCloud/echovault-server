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

	entsql "entgo.io/ent/dialect/sql"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	viper.SetDefault("grpc_port", 9090)
	viper.SetDefault("db_path", "data/echovault.db")
	viper.SetDefault("jwt_secret", "change-me-in-production")
	viper.AutomaticEnv()

	dbPath := viper.GetString("db_path")
	drv, err := entsql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_fk=1")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer drv.Close()

	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	ctx := context.Background()
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed to create schema: %v", err)
	}
	slog.Info("database migrated successfully")

	jwtSecret := viper.GetString("jwt_secret")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("grpc_port")))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(evgrpc.AuthInterceptor(jwtSecret)))
	evgrpc.RegisterAll(s, client, jwtSecret)

	ctx, cancel := context.WithCancel(context.Background())
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
