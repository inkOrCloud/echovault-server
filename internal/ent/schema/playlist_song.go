package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PlaylistSong holds the schema definition for the PlaylistSong entity.
type PlaylistSong struct {
	ent.Schema
}

// Fields returns the PlaylistSong fields.
func (PlaylistSong) Fields() []ent.Field {
	return []ent.Field{
		field.String("playlist_id"),
		field.String("song_id"),
		field.Int32("position").Default(0),
		field.Time("added_at"),
	}
}

// Indexes returns the PlaylistSong indexes.
func (PlaylistSong) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("playlist_id", "song_id").Unique(),
		index.Fields("playlist_id"),
	}
}
