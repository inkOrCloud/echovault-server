package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

const (
	songTitleMaxLen    = 512
	songArtistMaxLen   = 256
	songAlbumMaxLen    = 256
	songGenreMaxLen    = 128
	songFileNameMaxLen = 512
	songFileHashMaxLen = 64
	songMimeMaxLen     = 64
)

// Song holds the schema definition for the Song entity.
type Song struct {
	ent.Schema
}

// Fields returns the Song fields.
func (Song) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Immutable().Unique(),
		field.String("title").MaxLen(songTitleMaxLen),
		field.String("artist").MaxLen(songArtistMaxLen).Default(""),
		field.String("album").MaxLen(songAlbumMaxLen).Default(""),
		field.String("genre").MaxLen(songGenreMaxLen).Default(""),
		field.Int32("track_number").Default(0),
		field.Int32("disc_number").Default(1),
		field.Int32("duration_ms").Default(0),
		field.Int32("year").Default(0),
		field.String("file_name").MaxLen(songFileNameMaxLen).Default(""),
		field.Int64("file_size").Default(0),
		field.String("file_hash").MaxLen(songFileHashMaxLen).Default(""),
		field.String("mime_type").MaxLen(songMimeMaxLen).Default(""),
		field.Int32("bitrate").Default(0),
		field.Int32("sample_rate").Default(0),
		field.String("source").Default("local"),
		field.String("file_status").Default("local_only"),
		field.String("owner_id").Default(""),
		field.Int64("version").Default(1),
		field.Bool("is_deleted").Default(false),
		field.Time("created_at"),
		field.Time("updated_at"),
	}
}

// Indexes returns the Song indexes.
func (Song) Indexes() []ent.Index {
	return []ent.Index{
		// Lookup by file hash (for dedup check)
		index.Fields("file_hash"),

		// Filter by owner
		index.Fields("owner_id"),

		// Composite: list non-deleted songs per owner sorted by title
		index.Fields("owner_id", "is_deleted", "title"),

		// Search: filter non-deleted, sort by created_at
		index.Fields("is_deleted", "created_at"),

		// Version-based sync queries
		index.Fields("version"),
	}
}
