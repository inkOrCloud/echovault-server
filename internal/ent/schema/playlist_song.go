package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PlaylistSong struct {
	ent.Schema
}

func (PlaylistSong) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("playlist_id"),
		field.String("song_id"),
		field.Int32("position").Default(0),
		field.String("added_by").Default(""),
		field.Int64("version").Default(0),
		field.Time("created_at"),
	}
}

func (PlaylistSong) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("playlist_id", "song_id").Unique(),
		index.Fields("playlist_id", "position"),
	}
}
