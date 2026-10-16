package grpccodec

import (
	"bytes"
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type gogoMsg struct{ fastMsg }

func (m *gogoMsg) Marshal() ([]byte, error) {
	b := make([]byte, m.Size())
	n, err := m.MarshalToSizedBuffer(b)
	return b[:n], err
}
func (m *gogoMsg) String() string { return string(m.data) }
func (*gogoMsg) ProtoMessage()    {}

func TestSameBytesAsStd(t *testing.T) {
	m := &gogoMsg{fastMsg{data: []byte("hello")}}
	want, err := encoding.GetCodecV2(proto.Name).Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Codec.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Materialize(), want.Materialize()) {
		t.Fatal("codec bytes differ from the standard codec")
	}
}

func TestInterop(t *testing.T) {
	std := encoding.GetCodecV2(proto.Name)
	for _, c := range []struct {
		name           string
		server, client encoding.CodecV2
		ctype          string
	}{
		{"both", Codec, Codec, "application/grpc+proto"},
		{"server only", Codec, std, "application/grpc+proto"},
		{"client only", std, Codec, "application/grpc+proto"},
		{"neither", std, std, "application/grpc+proto"},
		{"client without options", Codec, nil, "application/grpc"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var ctype string
			lis := bufconn.Listen(1 << 20)
			s := grpc.NewServer(grpc.ForceServerCodecV2(c.server), grpc.UnknownServiceHandler(func(_ any, st grpc.ServerStream) error {
				md, _ := metadata.FromIncomingContext(st.Context())
				ctype = md.Get("content-type")[0]
				var m gogoMsg
				if err := st.RecvMsg(&m); err != nil {
					return err
				}
				return st.SendMsg(&m)
			}))
			go s.Serve(lis)
			defer s.Stop()
			opts := []grpc.DialOption{
				grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			}
			if c.client != nil {
				opts = append(opts, grpc.WithDefaultCallOptions(grpc.ForceCodecV2(c.client)))
			}
			conn, err := grpc.NewClient("passthrough:///bufnet", opts...)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			out := &gogoMsg{}
			if err := conn.Invoke(context.Background(), "/x.Y/Echo", &gogoMsg{fastMsg{data: []byte("ping")}}, out); err != nil {
				t.Fatal(err)
			}
			if string(out.data) != "ping" {
				t.Fatalf("got %q", out.data)
			}
			if ctype != c.ctype {
				t.Fatalf("content-type %q", ctype)
			}
		})
	}
}
