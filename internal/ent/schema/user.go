package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

const (
	userUsernameMaxLen    = 64
	userDisplayNameMaxLen = 128
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Fields returns the User fields.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable().Unique(),
		field.String("username").Unique().MaxLen(userUsernameMaxLen),
		field.String("display_name").MaxLen(userDisplayNameMaxLen).Default(""),
		field.String("password_hash"),
		field.String("role").Default("user"),
		field.Time("created_at"),
		field.Time("updated_at"),
	}
}

// Indexes returns the User indexes.
func (User) Indexes() []ent.Index {
	return nil
}
