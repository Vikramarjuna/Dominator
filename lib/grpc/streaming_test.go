package grpc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	liberrors "github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func chunks(data ...string) func() ([]byte, error) {
	return func() ([]byte, error) {
		if len(data) < 1 {
			return nil, io.EOF
		}
		chunk := []byte(data[0])
		data = data[1:]
		return chunk, nil
	}
}

func TestChunkReaderReadsAllChunks(t *testing.T) {
	reader := NewChunkReader(chunks("hello ", "", "world"))
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Fatalf("got %q", data)
	}
}

func TestChunkReaderSplitsLargeChunk(t *testing.T) {
	reader := NewChunkReader(chunks("abcdef"))
	buffer := make([]byte, 4)
	for _, want := range []string{"abcd", "ef"} {
		n, err := reader.Read(buffer)
		if err != nil {
			t.Fatal(err)
		}
		if string(buffer[:n]) != want {
			t.Fatalf("got %q, want %q", buffer[:n], want)
		}
	}
	if _, err := reader.Read(buffer); err != io.EOF {
		t.Fatalf("got %v, want io.EOF", err)
	}
}

func TestChunkReaderPropagatesError(t *testing.T) {
	failure := errors.New("broken stream")
	reader := NewChunkReader(func() ([]byte, error) {
		return nil, failure
	})
	if _, err := reader.Read(make([]byte, 4)); err != failure {
		t.Fatalf("got %v, want %v", err, failure)
	}
}

func TestStreamingConnDecodeEncode(t *testing.T) {
	var decoded, encoded interface{}
	conn := NewStreamingConn(context.Background(),
		func(v interface{}) error { decoded = v; return nil },
		func(v interface{}) error { encoded = v; return nil }, nil)
	request, reply := new(int), "reply"
	if err := conn.Decode(request); err != nil {
		t.Fatal(err)
	}
	if err := conn.Encode(reply); err != nil {
		t.Fatal(err)
	}
	if decoded != request || encoded != reply {
		t.Fatalf("decoded=%v encoded=%v", decoded, encoded)
	}
	if err := conn.Flush(); err != nil {
		t.Fatal(err)
	}
}

func TestStreamingConnAuthInformation(t *testing.T) {
	conn := NewStreamingConn(context.Background(), nil, nil, nil)
	if conn.GetAuthInformation() != nil || conn.Username() != "" {
		t.Fatal("unauthenticated conn reports a caller")
	}
	authInformation := &srpc.AuthInformation{Username: "alice"}
	ctx := ContextWithAuthInfo(context.Background(),
		&AuthInfo{authInformation: authInformation})
	conn = NewStreamingConn(ctx, nil, nil, nil)
	if conn.GetAuthInformation() != authInformation {
		t.Fatal("wrong authentication information")
	}
	if conn.Username() != "alice" {
		t.Fatalf("got username %q", conn.Username())
	}
}

func TestStreamingConnRead(t *testing.T) {
	conn := NewStreamingConn(context.Background(), nil, nil,
		bytes.NewReader([]byte("data")))
	buffer := make([]byte, 8)
	n, err := conn.Read(buffer)
	if err != nil || string(buffer[:n]) != "data" {
		t.Fatalf("got %q, %v", buffer[:n], err)
	}
}

func TestStreamingConnReadUnimplemented(t *testing.T) {
	conn := NewStreamingConn(context.Background(), nil, nil, nil)
	var unimplemented *liberrors.UnimplementedError
	if _, err := conn.Read(make([]byte, 1)); !errors.As(err, &unimplemented) {
		t.Fatalf("got %v", err)
	}
}
