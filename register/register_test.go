package register

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/test/bufconn"

	"gopkg.qsoa.cloud/grpccodec"
)

type probe struct{}

func (probe) Size() int                                { return 0 }
func (probe) MarshalToSizedBuffer([]byte) (int, error) { return 0, nil }
func (*probe) Unmarshal([]byte) error                  { return nil }
func (*probe) Reset()                                  {}

func TestInstalled(t *testing.T) {
	if encoding.GetCodecV2(proto.Name) != grpccodec.Codec {
		t.Fatal("grpccodec.Codec is not the registered proto codec")
	}
}

func TestRegistryRPC(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	s := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, st grpc.ServerStream) error {
		var m probe
		if err := st.RecvMsg(&m); err != nil {
			return err
		}
		return st.SendMsg(&m)
	}))
	go s.Serve(lis)
	defer s.Stop()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.Invoke(context.Background(), "/probe.Probe/Echo", &probe{}, &probe{}); err != nil {
		t.Fatal(err)
	}
}
