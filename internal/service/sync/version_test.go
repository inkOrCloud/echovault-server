package sync_test

import (
	"context"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	name := "file:sync_" + uuid.New().String() + "?mode=memory&cache=shared&_fk=1"
	drv, err := entsql.Open("sqlite3", name)
	require.NoError(t, err)
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	require.NoError(t, client.Schema.Create(context.Background()))
	return client
}

func TestNextVersion_StartsAt1(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	vt := sync.NewVersionTracker(client)
	ctx := context.Background()

	v, err := vt.Next(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), v)
}

func TestNextVersion_AfterWrite(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	vt := sync.NewVersionTracker(client)
	logger := sync.NewChangeLogger(client)
	ctx := context.Background()

	err := logger.Append(ctx, &syncpb.SyncChange{
		EntityType: testEntityType,
		EntityId:   uuid.New().String(),
		Action:     syncpb.SyncChange_ACTION_CREATE,
		Version:    1,
		DeviceId:   "dev-001",
	})
	require.NoError(t, err)

	v, err := vt.Next(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), v)
}

func TestCurrentVersion_Empty(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	vt := sync.NewVersionTracker(client)
	ctx := context.Background()

	cv, err := vt.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), cv)
}

func TestCurrentVersion_AfterWrites(t *testing.T) {
	t.Parallel()
	client := newTestClient(t)
	defer func() { _ = client.Close() }()
	vt := sync.NewVersionTracker(client)
	logger := sync.NewChangeLogger(client)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		err := logger.Append(ctx, &syncpb.SyncChange{
			EntityType: testEntityType,
			EntityId:   uuid.New().String(),
			Action:     syncpb.SyncChange_ACTION_CREATE,
			Version:    int64(i),
			DeviceId:   "dev-001",
		})
		require.NoError(t, err)
	}

	cv, err := vt.Current(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(3), cv)
}
