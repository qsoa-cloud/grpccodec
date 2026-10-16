package grpccodec

import (
	"bytes"
	"errors"
	"testing"

	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/mem"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type fastMsg struct{ data []byte }

func (m *fastMsg) Size() int {
	if len(m.data) == 0 {
		return 0
	}
	return protowire.SizeTag(1) + protowire.SizeBytes(len(m.data))
}

func (m *fastMsg) MarshalToSizedBuffer(b []byte) (int, error) {
	if len(m.data) == 0 {
		return 0, nil
	}
	out := protowire.AppendBytes(protowire.AppendTag(b[:0], 1, protowire.BytesType), m.data)
	return len(out), nil
}

func (m *fastMsg) Unmarshal(b []byte) error {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return protowire.ParseError(n)
		}
		if num != 1 || typ != protowire.BytesType {
			return errors.New("unexpected field")
		}
		v, k := protowire.ConsumeBytes(b[n:])
		if k < 0 {
			return protowire.ParseError(k)
		}
		m.data = append(m.data, v...)
		b = b[n+k:]
	}
	return nil
}

func (m *fastMsg) Reset() { m.data = nil }

type panicMsg struct{ fastMsg }

func (m *panicMsg) Unmarshal([]byte) error { panic("boom") }

type shortMsg struct{ fastMsg }

func (m *shortMsg) MarshalToSizedBuffer(b []byte) (int, error) {
	n, err := m.fastMsg.MarshalToSizedBuffer(b)
	return n - 1, err
}

func roundTrip(t testing.TB, in, out any) {
	data, err := Codec.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	defer data.Free()
	if err := Codec.Unmarshal(data, out); err != nil {
		t.Fatal(err)
	}
}

func TestName(t *testing.T) {
	if Codec.Name() != proto.Name {
		t.Fatalf("name %q, want %q", Codec.Name(), proto.Name)
	}
}

func TestFastRoundTrip(t *testing.T) {
	for _, n := range []int{0, 10, 64 << 10} {
		in := &fastMsg{data: bytes.Repeat([]byte{'x'}, n)}
		out := &fastMsg{}
		roundTrip(t, in, out)
		if !bytes.Equal(in.data, out.data) {
			t.Fatalf("size %d: got %d bytes", n, len(out.data))
		}
	}
}

func TestUnmarshalResets(t *testing.T) {
	out := &fastMsg{data: []byte("old")}
	roundTrip(t, &fastMsg{data: []byte("new")}, out)
	if string(out.data) != "new" {
		t.Fatalf("got %q, want %q", out.data, "new")
	}
}

func TestFallbackMatchesStd(t *testing.T) {
	std := encoding.GetCodecV2(proto.Name)
	in := wrapperspb.String("hello")
	want, err := std.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Codec.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Materialize(), want.Materialize()) {
		t.Fatal("fallback bytes differ from std codec")
	}
	out := &wrapperspb.StringValue{}
	if err := Codec.Unmarshal(got, out); err != nil || out.Value != "hello" {
		t.Fatalf("fallback unmarshal: %v %q", err, out.Value)
	}
}

func TestUnmarshalPanicIsError(t *testing.T) {
	data := mem.BufferSlice{mem.SliceBuffer([]byte{0x0a, 0x01, 'x'})}
	if err := Codec.Unmarshal(data, &panicMsg{}); err == nil {
		t.Fatal("want error from panicking Unmarshal")
	}
}

func TestMarshalShortWriteIsError(t *testing.T) {
	for _, n := range []int{10, 2 << 10} {
		if _, err := Codec.Marshal(&shortMsg{fastMsg{data: bytes.Repeat([]byte{'x'}, n)}}); err == nil {
			t.Errorf("size %d: want error from short MarshalToSizedBuffer", n)
		}
	}
}

func TestUnmarshalAddsNoAllocs(t *testing.T) {
	wire := []byte{0x0a, 0x03, 'a', 'b', 'c'}
	data := mem.BufferSlice{mem.SliceBuffer(wire)}
	m := &fastMsg{}
	direct := testing.AllocsPerRun(100, func() { m.Reset(); _ = m.Unmarshal(wire) })
	via := testing.AllocsPerRun(100, func() { _ = Codec.Unmarshal(data, m) })
	if via > direct {
		t.Fatalf("codec Unmarshal allocs %v, direct %v", via, direct)
	}
}

func TestMarshalAllocs(t *testing.T) {
	m := &fastMsg{data: []byte("abc")}
	n := testing.AllocsPerRun(100, func() {
		d, _ := Codec.Marshal(m)
		d.Free()
	})
	if n > 3 {
		t.Fatalf("Marshal allocs %v, want <= 3", n)
	}
}

var benchData = bytes.Repeat([]byte("Lorem ipsum dolor sit amet. "), 70)

func BenchmarkFast(b *testing.B) {
	in, out := &fastMsg{data: benchData}, &fastMsg{}
	b.ReportAllocs()
	for b.Loop() {
		roundTrip(b, in, out)
	}
}

func BenchmarkFallback(b *testing.B) {
	in, out := wrapperspb.Bytes(benchData), &wrapperspb.BytesValue{}
	b.ReportAllocs()
	for b.Loop() {
		roundTrip(b, in, out)
	}
}

func BenchmarkStd(b *testing.B) {
	std := encoding.GetCodecV2(proto.Name)
	in, out := wrapperspb.Bytes(benchData), &wrapperspb.BytesValue{}
	b.ReportAllocs()
	for b.Loop() {
		d, _ := std.Marshal(in)
		_ = std.Unmarshal(d, out)
		d.Free()
	}
}
