package register

import (
	"testing"

	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"

	"gopkg.qsoa.cloud/grpccodec"
)

func TestInstalled(t *testing.T) {
	if encoding.GetCodecV2(proto.Name) != grpccodec.Codec {
		t.Fatal("grpccodec.Codec is not the registered proto codec")
	}
}
