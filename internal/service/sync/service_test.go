package sync_test

import (
	"time"
	ent "github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"context"
	"github.com/google/uuid"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	syncsvc "github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	"github.com/stretchr/testify/require"
	"testing"
)

const testDeviceID = "dev-001"

func TestPushChanges_Empty(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := syncsvc.NewService(client)
	ctx := context.Background()

	resp, err := svc.PushChanges(ctx, testDeviceID, 0, nil)
	require.NoError(t, err)
	require.GreaterOrEqual(t, resp.ServerVersion, int64(0))
}

func TestPushChanges_SingleChange(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := syncsvc.NewService(client)
	ctx := context.Background()

	resp, err := svc.PushChanges(ctx, testDeviceID, 0, []*syncpb.SyncChange{{
		EntityType: testEntityType, EntityId: uuid.New().String(),
		Action: syncpb.SyncChange_ACTION_CREATE, DeviceId: testDeviceID,
	}})
	require.NoError(t, err)
	require.Equal(t, int64(1), resp.ServerVersion)
	require.Equal(t, int32(1), resp.AcceptedCount)
}

func TestPushChanges_Conflict(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := syncsvc.NewService(client)
	ctx := context.Background()
	songID := uuid.New().String()

	_, err := svc.PushChanges(ctx, testDeviceID, 0, []*syncpb.SyncChange{{
		EntityType: testEntityType, EntityId: songID, DeviceId: testDeviceID,
		Action: syncpb.SyncChange_ACTION_CREATE,
	}})
	require.NoError(t, err)

	_, err = svc.PushChanges(ctx, "dev-002", 0, []*syncpb.SyncChange{{
		EntityType: testEntityType, EntityId: songID, DeviceId: "dev-002",
		Action:  syncpb.SyncChange_ACTION_UPDATE,
		Version: 0,
	}})
	require.NoError(t, err)
}

func TestPullChanges(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := syncsvc.NewService(client)
	ctx := context.Background()

	_, err := svc.PushChanges(ctx, testDeviceID, 0, []*syncpb.SyncChange{{
		EntityType: testEntityType, EntityId: uuid.New().String(),
		Action: syncpb.SyncChange_ACTION_CREATE, DeviceId: testDeviceID,
	}})
	require.NoError(t, err)

	changes, err := svc.PullChanges(ctx, "dev-002", 0, 10)
	require.NoError(t, err)
	require.Len(t, changes, 1)
}

func TestPullChanges_OnlyNewer(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	svc := syncsvc.NewService(client)
	ctx := context.Background()

	for range 3 {
		_, err := svc.PushChanges(ctx, testDeviceID, 0, []*syncpb.SyncChange{{
			EntityType: testEntityType, EntityId: uuid.New().String(),
			Action: syncpb.SyncChange_ACTION_CREATE, DeviceId: testDeviceID,
		}})
		require.NoError(t, err)
	}

	changes, err := svc.PullChanges(ctx, "dev-002", 1, 10)
	require.NoError(t, err)
	require.Len(t, changes, 2)
}

func TestAckChanges(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	svc := syncsvc.NewService(client); ctx := context.Background()
	_, err := svc.PushChanges(ctx, testDeviceID, 0, []*syncpb.SyncChange{{EntityType: testEntityType, EntityId: uuid.New().String(), Action: syncpb.SyncChange_ACTION_CREATE, DeviceId: testDeviceID}})
	require.NoError(t, err)
	err = svc.AckChanges(ctx, testDeviceID, 1)
	require.NoError(t, err)
}
func TestSyncEntToProto(t *testing.T) {
	t.Parallel()
	pb := syncsvc.EntToProto(&ent.SyncLog{ID:"l",DeviceID:"d",EntityType:"song",EntityID:"s1",Action:"create",Version:1,Timestamp:time.Now()})
	if pb.GetEntityType() != "song" { t.Errorf("got %q", pb.GetEntityType()) }
	if pb.GetAction() != syncpb.SyncChange_ACTION_CREATE { t.Errorf("got %v", pb.GetAction()) }
}
func TestSyncEntToProto_Nil(t *testing.T) {
	t.Parallel()
	if pb := syncsvc.EntToProto(nil); pb != nil { t.Error("should be nil") }
}
func TestProtoActionToEnt(t *testing.T) {
	t.Parallel()
	if syncsvc.ProtoActionToEnt(syncpb.SyncChange_ACTION_CREATE) != "create" { t.Error("create mismatch") }
	if syncsvc.ProtoActionToEnt(syncpb.SyncChange_ACTION_UPDATE) != "update" { t.Error("update mismatch") }
	if syncsvc.ProtoActionToEnt(syncpb.SyncChange_ACTION_DELETE) != "delete" { t.Error("delete mismatch") }
	if syncsvc.ProtoActionToEnt(syncpb.SyncChange_ACTION_UNSPECIFIED) != "unknown" { t.Error("unspec mismatch") }
}
func TestChangeLogger_GetByEntity(t *testing.T) {
	t.Parallel(); client := newTestClient(t); defer client.Close()
	logger := syncsvc.NewChangeLogger(client); ctx := context.Background()
	_, err := logger.GetByEntity(ctx, testEntityType, "x")
	require.Error(t, err)
	logger.Append(ctx, &syncpb.SyncChange{EntityType: testEntityType, EntityId: "e1", Action: syncpb.SyncChange_ACTION_CREATE, Version: 1, DeviceId: "d1"})
	record, err := logger.GetByEntity(ctx, testEntityType, "e1")
	require.NoError(t, err)
	require.Equal(t, "e1", record.EntityID)
}
func TestNotifier_BufferFull(t *testing.T) {
	t.Parallel()
	n := syncsvc.NewNotifier()
	_ = n.Subscribe("d"); defer n.Unsubscribe("d")
	for i := 0; i < 32; i++ { n.Notify(&syncpb.ChangeNotification{NewVersion: int64(i)}) }
}
