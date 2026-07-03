package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent/synclog"
)

// ChangeLogger persists sync changes to the database.
type ChangeLogger struct {
	client *ent.Client
}

// NewChangeLogger creates a new ChangeLogger.
func NewChangeLogger(client *ent.Client) *ChangeLogger {
	return &ChangeLogger{client: client}
}

// Append records a sync change.
func (l *ChangeLogger) Append(ctx context.Context, change *syncpb.SyncChange) error {
	_, err := l.client.SyncLog.Create().
		SetID(uuid.New().String()).
		SetDeviceID(change.GetDeviceId()).
		SetEntityType(change.GetEntityType()).
		SetEntityID(change.GetEntityId()).
		SetAction(change.GetAction().String()).
		SetVersion(change.GetVersion()).
		SetData(change.GetData()).
		SetTimestamp(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("append sync log: %w", err)
	}
	return nil
}

// QuerySince retrieves changes since a given version.
func (l *ChangeLogger) QuerySince(ctx context.Context, sinceVersion int64, limit int) ([]*ent.SyncLog, error) {
	return l.client.SyncLog.Query().
		Where(synclog.VersionGT(sinceVersion)).
		Order(ent.Asc(synclog.FieldVersion)).
		Limit(limit).
		All(ctx)
}

// Ack marks changes up to a version as acknowledged.
func (l *ChangeLogger) Ack(ctx context.Context, version int64) error {
	_, err := l.client.SyncLog.Update().
		Where(synclog.VersionLTE(version)).
		SetAcked(true).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("ack sync log: %w", err)
	}
	return nil
}

// GetByEntity retrieves the latest sync log for an entity.
func (l *ChangeLogger) GetByEntity(ctx context.Context, entityType, entityID string) (*ent.SyncLog, error) {
	return l.client.SyncLog.Query().
		Where(synclog.EntityType(entityType), synclog.EntityID(entityID)).
		Order(ent.Desc(synclog.FieldVersion)).
		First(ctx)
}
