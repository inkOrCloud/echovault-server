// Package main is the entry point for the EchoVault server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/rest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/song"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/metadata"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/storage"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const (
	defaultGRPCPort     = 9090
	defaultRESTPort     = 9091
	shutdownTimeoutSecs = 5
)

// Sentinel errors for overflow checks.
var (
	ErrTrackNumberOverflow = errors.New("track number overflows int32")
	ErrDiscNumberOverflow  = errors.New("disc number overflows int32")
	ErrYearOverflow        = errors.New("year overflows int32")
)

// songUpdaterAdapter bridges the REST handler's SongUpdater to SongService.
type songUpdaterAdapter struct {
	svc *song.Service
}

func (a *songUpdaterAdapter) UpdateFromScan(songID string, meta *metadata.AudioMetadata, fileSize int64) error {
	if meta.TrackNumber > math.MaxInt32 || meta.TrackNumber < math.MinInt32 {
		return ErrTrackNumberOverflow
	}
	if meta.DiscNumber > math.MaxInt32 || meta.DiscNumber < math.MinInt32 {
		return ErrDiscNumberOverflow
	}
	if meta.Year > math.MaxInt32 || meta.Year < math.MinInt32 {
		return ErrYearOverflow
	}
	err := a.svc.UpdateFromScan(context.Background(), songID,
		meta.Title, meta.Artist, meta.Album, meta.Genre,
		int32(meta.TrackNumber), int32(meta.DiscNumber), int32(meta.Year),
		meta.FileHash, meta.FileName, meta.MIMEType, fileSize)
	if err != nil {
		return fmt.Errorf("update from scan: %w", err)
	}
	return nil
}

func setupGRPC(client *ent.Client, jwtSecret string) (*grpc.Server, net.Listener) {
	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", fmt.Sprintf(":%d", viper.GetInt("grpc_port")))
	if err != nil {
		log.Fatalf("failed to listen gRPC: %v", err)
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(evgrpc.AuthInterceptor(jwtSecret)))
	evgrpc.RegisterAll(s, client, jwtSecret)
	return s, lis
}

func setupREST(songSvc *song.Service) *http.Server {
	storageSvc, err := storage.NewStorage(
		viper.GetString("storage_type"),
		viper.GetString("storage_path"),
	)
	if err != nil {
		log.Fatalf("failed to init storage: %v", err)
	}
	songUpdater := &songUpdaterAdapter{svc: songSvc}
	restHandler := rest.NewHandler(storageSvc, songUpdater)
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", viper.GetInt("rest_port")),
		Handler:           restHandler,
		ReadHeaderTimeout: 10 * time.Second, //nolint:mnd
	}
}

func main() {
	viper.SetDefault("grpc_port", defaultGRPCPort)
	viper.SetDefault("rest_port", defaultRESTPort)
	viper.SetDefault("db_path", "data/echovault.db")
	viper.SetDefault("jwt_secret", "change-me-in-production")
	viper.SetDefault("storage_type", "local")
	viper.SetDefault("storage_path", "data/files")
	viper.AutomaticEnv()

	dbPath := viper.GetString("db_path")
	drv, err := entsql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_fk=1")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	client := ent.NewClient(ent.Driver(drv))

	ctx := context.Background()
	err = client.Schema.Create(ctx)
	if err != nil {
		_ = drv.Close()
		_ = client.Close()
		log.Fatalf("failed to create schema: %v", err)
	}
	defer func() { _ = drv.Close() }()
	defer func() { _ = client.Close() }()
	slog.Info("database migrated successfully")

	jwtSecret := viper.GetString("jwt_secret")
	s, lis := setupGRPC(client, jwtSecret)

	songSvc := song.NewService(client)
	ginServer := setupREST(songSvc)

	// Graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutting down servers...")
		cancel()
		s.GracefulStop()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeoutSecs*time.Second)
		defer shutdownCancel()
		_ = ginServer.Shutdown(shutdownCtx)
	}()

	go func() {
		slog.Info("starting REST file server", "port", viper.GetInt("rest_port"))
		err := ginServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("REST server error: %v", err)
		}
	}()

	slog.Info("starting EchoVault server", "port", viper.GetInt("grpc_port"))
	err = s.Serve(lis)
	if err != nil {
		cancel()
		slog.Error("gRPC server error", "error", err)
		os.Exit(1) //nolint:gocritic // cancel() is called explicitly above
	}
}
