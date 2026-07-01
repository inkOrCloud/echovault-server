package grpc_test

import (
	"context"
	"io"
	"net"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	syncsvc "github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

func newSyncTestServer(t *testing.T) (syncpb.SyncServiceClient, func()) {
	t.Helper()
	name := "file:sync_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	svc := syncsvc.NewService(client)
	handler := evgrpc.NewSyncHandler(svc)

	s := grpc.NewServer()
	syncpb.RegisterSyncServiceServer(s, handler)
	lis, _ := net.Listen("tcp", "127.0.0.1:0")
	go s.Serve(lis)
	conn, _ := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	syncClient := syncpb.NewSyncServiceClient(conn)

	return syncClient, func() { conn.Close(); s.GracefulStop(); client.Close() }
}

func TestSyncPushHandler(t *testing.T) {
	client, cleanup := newSyncTestServer(t)
	defer cleanup()

	resp, err := client.PushChanges(context.Background(), &syncpb.PushChangesRequest{
		DeviceId: "dev-001",
		Changes: []*syncpb.SyncChange{{
			EntityType: "song",
			EntityId:   uuid.New().String(),
			Action:     syncpb.SyncChange_ACTION_CREATE,
		}},
	})
	if err != nil {
		t.Fatalf("PushChanges RPC error = %v", err)
	}
	if resp.ServerVersion < 1 {
		t.Errorf("ServerVersion = %d, want >=1", resp.ServerVersion)
	}
}

func TestSyncPullHandler(t *testing.T) {
	client, cleanup := newSyncTestServer(t)
	defer cleanup()

	client.PushChanges(context.Background(), &syncpb.PushChangesRequest{
		DeviceId: "dev-001",
		Changes: []*syncpb.SyncChange{{
			EntityType: "song", EntityId: uuid.New().String(),
			Action: syncpb.SyncChange_ACTION_CREATE,
		}},
	})

	stream, err := client.PullChanges(context.Background(), &syncpb.PullChangesRequest{
		DeviceId: "dev-002", SinceVersion: 0,
	})
	if err != nil {
		t.Fatalf("PullChanges RPC error = %v", err)
	}

	count := 0
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("PullChanges stream error = %v", err)
		}
		count++
	}
	if count != 1 {
		t.Errorf("PullChanges count = %d, want 1", count)
	}
}
