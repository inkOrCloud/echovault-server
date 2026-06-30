package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type Playlist struct {
	ent.Schema
}

func (Playlist) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("name").MaxLen(256),
		field.String("description").MaxLen(1024).Default(""),
		field.String("cover_url").Default(""),
		field.String("type").Default("user"),
		field.String("owner_id"),
		field.Bool("is_public").Default(false),
		field.Int32("song_count").Default(0),
		field.Bool("is_deleted").Default(false),
		field.Int64("version").Default(0),
		field.Time("created_at"),
		field.Time("updated_at"),
	}
}
