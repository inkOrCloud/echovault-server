package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SyncLog struct {
	ent.Schema
}

func (SyncLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("device_id"),
		field.String("entity_type"),
		field.String("entity_id"),
		field.String("action"),
		field.Int64("version"),
		field.Bytes("data").Optional(),
		field.Time("timestamp"),
		field.Bool("acked").Default(false),
	}
}

func (SyncLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("device_id", "version"),
		index.Fields("acked"),
	}
}
