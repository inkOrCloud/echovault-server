package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Lyric struct {
	ent.Schema
}

func (Lyric) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("song_id"),
		field.Text("content"),
		field.String("type").Default("original"),
		field.String("language").Default(""),
		field.Int32("offset_ms").Default(0),
		field.String("source").Default("manual"),
		field.Bool("is_deleted").Default(false),
		field.Int64("version").Default(0),
		field.Time("created_at"),
		field.Time("updated_at"),
	}
}

func (Lyric) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("song_id"),
	}
}
