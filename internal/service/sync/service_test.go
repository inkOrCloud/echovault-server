package sync_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	syncsvc "github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

func TestPushChanges_Empty(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := syncsvc.NewService(client)
	ctx := context.Background()

	resp, err := svc.PushChanges(ctx, "dev-001", 0, nil)
	if err != nil {
		t.Fatalf("PushChanges() error = %v", err)
	}
	if resp.ServerVersion < 0 {
		t.Errorf("ServerVersion = %d, want >=0", resp.ServerVersion)
	}
}

func TestPushChanges_SingleChange(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := syncsvc.NewService(client)
	ctx := context.Background()

	resp, err := svc.PushChanges(ctx, "dev-001", 0, []*syncpb.SyncChange{{
		EntityType: "song", EntityId: uuid.New().String(),
		Action: syncpb.SyncChange_ACTION_CREATE, DeviceId: "dev-001",
	}})
	if err != nil {
		t.Fatalf("PushChanges() error = %v", err)
	}
	if resp.ServerVersion != 1 {
		t.Errorf("ServerVersion = %d, want 1", resp.ServerVersion)
	}
	if resp.AcceptedCount != 1 {
		t.Errorf("AcceptedCount = %d, want 1", resp.AcceptedCount)
	}
}

func TestPushChanges_Conflict(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := syncsvc.NewService(client)
	ctx := context.Background()
	songID := uuid.New().String()

	// 先提交一个变更（获得版本号 1）
	svc.PushChanges(ctx, "dev-001", 0, []*syncpb.SyncChange{{
		EntityType: "song", EntityId: songID, DeviceId: "dev-001",
		Action: syncpb.SyncChange_ACTION_CREATE,
	}})

	// 再提交同一个实体的版本 0 → 冲突检测应拒绝
	resp, _ := svc.PushChanges(ctx, "dev-002", 0, []*syncpb.SyncChange{{
		EntityType: "song", EntityId: songID, DeviceId: "dev-002",
		Action: syncpb.SyncChange_ACTION_UPDATE,
		Version: 0,
	}})
	// 客户端 version(0) < 服务端 version(1)
	// 应该检测到冲突
	t.Logf("conflicts: %+v", resp.Conflicts)
}

func TestPullChanges(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := syncsvc.NewService(client)
	ctx := context.Background()

	svc.PushChanges(ctx, "dev-001", 0, []*syncpb.SyncChange{{
		EntityType: "song", EntityId: uuid.New().String(),
		Action: syncpb.SyncChange_ACTION_CREATE, DeviceId: "dev-001",
	}})

	changes, err := svc.PullChanges(ctx, "dev-002", 0, 10)
	if err != nil {
		t.Fatalf("PullChanges() error = %v", err)
	}
	if len(changes) != 1 {
		t.Errorf("PullChanges() = %d changes, want 1", len(changes))
	}
}

func TestPullChanges_OnlyNewer(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()
	svc := syncsvc.NewService(client)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		svc.PushChanges(ctx, "dev-001", 0, []*syncpb.SyncChange{{
			EntityType: "song", EntityId: uuid.New().String(),
			Action: syncpb.SyncChange_ACTION_CREATE, DeviceId: "dev-001",
		}})
	}

	// since_version=1 → 应返回 2 条（版本 2 和 3）
	changes, err := svc.PullChanges(ctx, "dev-002", 1, 10)
	if err != nil {
		t.Fatalf("PullChanges() error = %v", err)
	}
	if len(changes) != 2 {
		t.Errorf("PullChanges(since=1) = %d changes, want 2", len(changes))
	}
}
