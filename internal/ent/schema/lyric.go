package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Lyric holds the schema definition for the Lyric entity.
type Lyric struct {
	ent.Schema
}

// Fields returns the Lyric fields.
func (Lyric) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable().Unique(),
		field.String("song_id"),
		field.Text("content"),
		field.String("type").Default("original"),
		field.String("language").Default(""),
		field.Int32("offset_ms").Default(0),
		field.String("source").Default(""),
		field.Int64("version").Default(1),
		field.Time("created_at"),
		field.Time("updated_at"),
	}
}

// Indexes returns the Lyric indexes.
func (Lyric) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("song_id", "type", "language").Unique(),
	}
}
