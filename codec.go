// Package grpccodec provides a gRPC codec that marshals gogo-generated messages through their generated methods.
package grpccodec

import (
	"fmt"

	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/mem"
)

type fastMessage interface {
	Size() int
	MarshalToSizedBuffer([]byte) (int, error)
	Unmarshal([]byte) error
	Reset()
}

type codec struct{ std encoding.CodecV2 }

// Codec handles messages with gogo-generated Size/MarshalToSizedBuffer/Unmarshal methods directly and delegates the rest to the standard proto codec.
var Codec encoding.CodecV2 = &codec{std: encoding.GetCodecV2(proto.Name)}

func (c *codec) Name() string { return proto.Name }

func (c *codec) Marshal(v any) (mem.BufferSlice, error) {
	m, ok := v.(fastMessage)
	if !ok {
		return c.std.Marshal(v)
	}
	size := m.Size()
	if mem.IsBelowBufferPoolingThreshold(size) {
		buf := make([]byte, size)
		if _, err := m.MarshalToSizedBuffer(buf); err != nil {
			return nil, err
		}
		return mem.BufferSlice{mem.SliceBuffer(buf)}, nil
	}
	pool := mem.DefaultBufferPool()
	buf := pool.Get(size)
	if _, err := m.MarshalToSizedBuffer(*buf); err != nil {
		pool.Put(buf)
		return nil, err
	}
	return mem.BufferSlice{mem.NewBuffer(buf, pool)}, nil
}

func (c *codec) Unmarshal(data mem.BufferSlice, v any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("codec: unmarshal %T: %v", v, r)
		}
	}()
	m, ok := v.(fastMessage)
	if !ok {
		return c.std.Unmarshal(data, v)
	}
	buf := data.MaterializeToBuffer(mem.DefaultBufferPool())
	defer buf.Free()
	m.Reset()
	return m.Unmarshal(buf.ReadOnlyData())
}
