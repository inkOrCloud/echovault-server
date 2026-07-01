package grpc

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/inkOrCloud/EchoVault/echovault-server/internal/ent"
)

func RegisterAll(s *grpc.Server, client *ent.Client) {
	_ = client // 后续阶段在此注册具体服务

	reflection.Register(s)
}
