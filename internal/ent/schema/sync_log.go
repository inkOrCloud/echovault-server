package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SyncLog holds the schema definition for the SyncLog entity.
type SyncLog struct {
	ent.Schema
}

// Fields returns the SyncLog fields.
func (SyncLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("device_id"),
		field.String("entity_type"),
		field.String("entity_id"),
		field.String("action"),
		field.Int64("version"),
		field.Bytes("data").Optional(),
		field.Bool("acked").Default(false),
		field.Time("timestamp"),
	}
}

// Indexes returns the SyncLog indexes.
func (SyncLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("version"),
		index.Fields("entity_type", "entity_id"),
		index.Fields("device_id"),
	}
}
