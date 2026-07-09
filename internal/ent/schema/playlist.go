package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

const (
	playlistNameMaxLen        = 256
	playlistDescriptionMaxLen = 1024
)

// Playlist holds the schema definition for the Playlist entity.
type Playlist struct {
	ent.Schema
}

// Fields returns the Playlist fields.
func (Playlist) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable().Unique(),
		field.String("name").MaxLen(playlistNameMaxLen),
		field.String("description").MaxLen(playlistDescriptionMaxLen).Default(""),
		field.String("cover_url").Default(""),
		field.String("owner_id"),
		field.Bool("is_public").Default(false),
		field.Int("song_count").Default(0),
		field.Int64("version").Default(1),
		field.Time("created_at"),
		field.Time("updated_at"),
	}
}

// Indexes returns the Playlist indexes.
func (Playlist) Indexes() []ent.Index {
	return []ent.Index{
		// Filter by owner
		index.Fields("owner_id"),

		// Composite: list playlists per owner sorted by name
		index.Fields("owner_id", "name"),

		// Public playlist discovery
		index.Fields("is_public"),
	}
}
