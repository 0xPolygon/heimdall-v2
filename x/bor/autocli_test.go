package bor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	borapi "github.com/0xPolygon/heimdall-v2/api/heimdallv2/bor"
)

func TestAutoCLIOptionsSkipMapResponses(t *testing.T) {
	skipped := make(map[string]bool)
	for _, opt := range (AppModule{}).AutoCLIOptions().Query.RpcCommandOptions {
		skipped[opt.RpcMethod] = opt.Skip
	}

	methods := borapi.File_heimdallv2_bor_query_proto.Services().ByName("Query").Methods()
	for i := 0; i < methods.Len(); i++ {
		method := methods.Get(i)
		require.Equal(t, hasMapField(method.Output()), skipped[string(method.Name())], method.Name())
	}
}

func hasMapField(msg protoreflect.MessageDescriptor) bool {
	fields := msg.Fields()
	for i := 0; i < fields.Len(); i++ {
		if fields.Get(i).IsMap() {
			return true
		}
	}
	return false
}
