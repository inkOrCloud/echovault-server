package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Song struct {
	ent.Schema
}

func (Song) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("title").MaxLen(512),
		field.String("artist").MaxLen(256).Default(""),
		field.String("album").MaxLen(256).Default(""),
		field.String("genre").MaxLen(128).Default(""),
		field.Int32("track_number").Default(0),
		field.Int32("disc_number").Default(1),
		field.Int32("duration_ms").Default(0),
		field.Int32("year").Default(0),
		field.String("file_name").MaxLen(512).Default(""),
		field.Int64("file_size").Default(0),
		field.String("file_hash").MaxLen(64).Default(""),
		field.String("mime_type").MaxLen(64).Default(""),
		field.Int32("bitrate").Default(0),
		field.Int32("sample_rate").Default(0),
		field.String("source").Default("local"),
		field.String("file_status").Default("local_only"),
		field.String("owner_id").Default(""),
		field.Bool("is_deleted").Default(false),
		field.Int64("version").Default(0),
		field.Time("created_at"),
		field.Time("updated_at"),
	}
}

func (Song) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("file_hash"),
		index.Fields("owner_id"),
		index.Fields("title", "artist"),
	}
}
