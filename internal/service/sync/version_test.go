package sync_test

import (
	"context"
	"testing"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/enttest"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

func newTestClient(t *testing.T) *ent.Client {
	t.Helper()
	drv, err := entsql.Open("sqlite3", "file:sync?mode=memory&cache=shared&_fk=1")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	client := enttest.NewClient(t, enttest.WithOptions(ent.Driver(drv)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return client
}

func TestNextVersion_StartsAt1(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	vt := sync.NewVersionTracker(client)
	ctx := context.Background()

	v, err := vt.Next(ctx)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if v != 1 {
		t.Errorf("Next() = %d, want 1 (first call, no records)", v)
	}
}

func TestNextVersion_AfterWrite(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	vt := sync.NewVersionTracker(client)
	logger := sync.NewChangeLogger(client)
	ctx := context.Background()

	// 写入一条变更（分配版本号 1）
	logger.Append(ctx, &syncpb.SyncChange{
		EntityType: "song",
		EntityId:   uuid.New().String(),
		Action:     syncpb.SyncChange_ACTION_CREATE,
		Version:    1,
		DeviceId:   "dev-001",
	})

	// 下次 Next 应该返回 2
	v, err := vt.Next(ctx)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	if v != 2 {
		t.Errorf("Next() = %d, want 2", v)
	}
}

func TestCurrentVersion_Empty(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	vt := sync.NewVersionTracker(client)
	ctx := context.Background()

	cv, err := vt.Current(ctx)
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if cv != 0 {
		t.Errorf("Current() = %d, want 0 (no records)", cv)
	}
}

func TestCurrentVersion_AfterWrites(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	vt := sync.NewVersionTracker(client)
	logger := sync.NewChangeLogger(client)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		logger.Append(ctx, &syncpb.SyncChange{
			EntityType: "song",
			EntityId:   uuid.New().String(),
			Action:     syncpb.SyncChange_ACTION_CREATE,
			Version:    int64(i),
			DeviceId:   "dev-001",
		})
	}

	cv, err := vt.Current(ctx)
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if cv != 3 {
		t.Errorf("Current() = %d, want 3", cv)
	}
}
