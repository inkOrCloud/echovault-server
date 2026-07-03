package sync_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	syncsvc "github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
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
		Action: syncpb.SyncChange_ACTION_UPDATE,
		Version:    0,
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
