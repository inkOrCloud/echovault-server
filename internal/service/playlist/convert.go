// Package playlist provides playlist management operations.
package playlist

import (
	playlistpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/playlist/v1"
	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
	"github.com/inkOrCloud/EchoVault/echovault-server/pkg/convert"
)

// EntToProto converts an ent Playlist to a proto Playlist.
func EntToProto(p *ent.Playlist) *playlistpb.Playlist {
	if p == nil {
		return nil
	}
	return &playlistpb.Playlist{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CoverUrl:    p.CoverURL,
		OwnerId:     p.OwnerID,
		IsPublic:    p.IsPublic,
		SongCount:   int32(p.SongCount), //nolint:gosec
		Version:     p.Version,
		CreatedAt:   convert.PTime(p.CreatedAt),
		UpdatedAt:   convert.PTime(p.UpdatedAt),
	}
}
