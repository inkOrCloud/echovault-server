package lyric

import (
	lyricpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/lyric/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/convert"
)

func EntToProto(l *ent.Lyric) *lyricpb.Lyric {
	if l == nil {
		return nil
	}
	return &lyricpb.Lyric{
		SongId:    l.SongID,
		Content:   l.Content,
		Type:      lyricTypeToProto(l.Type),
		Language:  l.Language,
		OffsetMs:  l.OffsetMs,
		Source:    lyricSourceToProto(l.Source),
		Version:   l.Version,
		UpdatedAt: convert.PTime(l.UpdatedAt),
	}
}

func lyricTypeToProto(s string) lyricpb.Lyric_Type {
	switch s {
	case "original":
		return lyricpb.Lyric_TYPE_ORIGINAL
	case "translation":
		return lyricpb.Lyric_TYPE_TRANSLATION
	case "phonetic":
		return lyricpb.Lyric_TYPE_PHONETIC
	default:
		return lyricpb.Lyric_TYPE_UNSPECIFIED
	}
}

func lyricSourceToProto(s string) lyricpb.Lyric_Source {
	switch s {
	case "embedded":
		return lyricpb.Lyric_SOURCE_EMBEDDED
	case "manual":
		return lyricpb.Lyric_SOURCE_MANUAL
	case "fetched":
		return lyricpb.Lyric_SOURCE_FETCHED
	default:
		return lyricpb.Lyric_SOURCE_UNSPECIFIED
	}
}
