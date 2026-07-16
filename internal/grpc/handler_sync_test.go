package grpc_test

import (
	require "github.com/stretchr/testify/require"
	"context"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	evgrpc "github.com/inkOrCloud/EchoVault/echovault-server/internal/grpc"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"testing"
)

func newSyncTestServer(t *testing.T) (syncpb.SyncServiceClient, func()) {
	t.Helper()
	name := "file:sync_handler_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	err = client.Schema.Create(context.Background())
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}
	svc := sync.NewService(client)
	handler := evgrpc.NewSyncHandler(svc)
	s := grpc.NewServer()
	syncpb.RegisterSyncServiceServer(s, handler)
	lc := net.ListenConfig{}
	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = s.Serve(lis) }()
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	c := syncpb.NewSyncServiceClient(conn)
	return c, func() { _ = conn.Close(); s.GracefulStop() }
}

func TestSyncPushHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newSyncTestServer(t)
	defer cleanup()

	resp, err := c.PushChanges(context.Background(), &syncpb.PushChangesRequest{
		DeviceId: testDeviceID,
		Changes: []*syncpb.SyncChange{{
			EntityType: "song",
			EntityId:   uuid.New().String(),
			Action:     syncpb.SyncChange_ACTION_CREATE,
			DeviceId:   testDeviceID,
		}},
	})
	if err != nil {
		t.Fatalf("PushChanges RPC error = %v", err)
	}
	if resp.GetServerVersion() != 1 {
		t.Errorf("ServerVersion = %d, want 1", resp.GetServerVersion())
	}
}

func TestSyncAckChangesHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newSyncTestServer(t)
	defer cleanup()

	push, err := c.PushChanges(context.Background(), &syncpb.PushChangesRequest{
		DeviceId: "dev-ack",
		Changes: []*syncpb.SyncChange{{
			EntityType: "song",
			EntityId:   uuid.New().String(),
			Action:     syncpb.SyncChange_ACTION_CREATE,
			DeviceId:   "dev-ack",
		}},
	})
	require.NoError(t, err)

	_, err = c.AckChanges(context.Background(), &syncpb.AckChangesRequest{
		DeviceId:     "dev-ack",
		AckedVersion: push.GetServerVersion(),
	})
	require.NoError(t, err)
}

func TestSyncEmptyChangesHandler(t *testing.T) {
	t.Parallel()
	c, cleanup := newSyncTestServer(t)
	defer cleanup()

	push, err := c.PushChanges(context.Background(), &syncpb.PushChangesRequest{
		DeviceId: "dev-empty",
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, push.GetServerVersion(), int64(0))
}
