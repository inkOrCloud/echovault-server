package sync

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/synclog"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

type ChangeLogger struct {
	client *ent.Client
}

func NewChangeLogger(client *ent.Client) *ChangeLogger {
	return &ChangeLogger{client: client}
}

func (l *ChangeLogger) Append(ctx context.Context, change *syncpb.SyncChange) error {
	_, err := l.client.SyncLog.Create().
		SetID(uuid.New().String()).
		SetDeviceID(change.DeviceId).
		SetEntityType(change.EntityType).
		SetEntityID(change.EntityId).
		SetAction(change.Action.String()).
		SetVersion(change.Version).
		SetData(change.Data).
		SetTimestamp(time.Now()).
		Save(ctx)
	return err
}

func (l *ChangeLogger) QuerySince(ctx context.Context, sinceVersion int64, limit int) ([]*ent.SyncLog, error) {
	return l.client.SyncLog.Query().
		Where(synclog.VersionGT(sinceVersion)).
		Order(ent.Asc(synclog.FieldVersion)).
		Limit(limit).
		All(ctx)
}

func (l *ChangeLogger) Ack(ctx context.Context, version int64) error {
	_, err := l.client.SyncLog.Update().
		Where(synclog.VersionLTE(version)).
		SetAcked(true).
		Save(ctx)
	return err
}

func (l *ChangeLogger) GetByEntity(ctx context.Context, entityType, entityID string) (*ent.SyncLog, error) {
	return l.client.SyncLog.Query().
		Where(synclog.EntityType(entityType), synclog.EntityID(entityID)).
		Order(ent.Desc(synclog.FieldVersion)).
		First(ctx)
}
