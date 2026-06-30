package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("username").Unique().MaxLen(64),
		field.String("display_name").MaxLen(128).Default(""),
		field.String("password_hash"),
		field.String("role").Default("user"),
		field.Bool("is_deleted").Default(false),
		field.Int64("version").Default(0),
		field.Time("created_at"),
		field.Time("updated_at"),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("username"),
	}
}
