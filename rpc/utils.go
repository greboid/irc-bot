package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc/metadata"
	"strings"
)

func CtxWithToken(ctx context.Context, scheme string, token string) context.Context {
	md := metadata.Pairs("authorization", fmt.Sprintf("%s %v", scheme, token))
	nCtx := metautils.NiceMD(md).ToOutgoing(ctx)
	return nCtx
}

func CtxWithTokenAndPath(ctx context.Context, scheme string, token string, path string) context.Context {
	md := metadata.Pairs("authorization", fmt.Sprintf("%s %v", scheme, token))
	md.Append("path", path)
	nCtx := metautils.NiceMD(md).ToOutgoing(ctx)
	return nCtx
}

func ParsePluginString(pluginString string) (plugins []Plugin, err error) {
	for _, value := range strings.Split(pluginString, ",") {
		if len(value) == 0 {
			break
		}
		pluginString := strings.Split(value, "=")
		if len(pluginString) != 2 {
			return nil, errors.New("invalid plugin definition")
		}
		plugins = append(plugins, Plugin{Name: pluginString[0], Token: pluginString[1]})
	}
	return
}

type Plugin struct {
	Name  string
	Token string
}
