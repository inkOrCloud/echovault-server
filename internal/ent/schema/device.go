// Package schema defines the ent schema types for EchoVault.
package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

const deviceNameMaxLen = 128

// Device holds the schema definition for the Device entity.
type Device struct {
	ent.Schema
}

// Fields returns the Device fields.
func (Device) Fields() []ent.Field {
	return []ent.Field{
		field.String("device_id").Unique().Immutable(),
		field.String("device_name").MaxLen(deviceNameMaxLen).Default(""),
		field.String("platform").Default(""),
		field.String("os_version").Default(""),
		field.String("client_version").Default(""),
		field.String("user_id"),
		field.Time("last_sync_at").Optional().Nillable(),
		field.String("sync_state").Default("pending"),
		field.Time("created_at"),
		field.Time("updated_at"),
	}
}

// Indexes returns the Device indexes.
func (Device) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
	}
}
