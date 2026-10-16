// Package register installs grpccodec.Codec as the process-wide gRPC "proto" codec when imported.
package register

import (
	"google.golang.org/grpc/encoding"

	"gopkg.qsoa.cloud/grpccodec"
)

func init() { encoding.RegisterCodecV2(grpccodec.Codec) }
