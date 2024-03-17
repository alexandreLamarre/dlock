package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	embeddedapi "github.com/alexandreLamarre/dlock/internal/embedded/api"
	"github.com/alexandreLamarre/dlock/pkg/node"
	"github.com/hashicorp/raft"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type mutexBackend struct {
	mu sync.RWMutex
	lg *slog.Logger
}

var _ raft.FSM = (*mutexBackend)(nil)

func (m *mutexBackend) Apply(l *raft.Log) interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	data := l.Data
	m.lg.Info(fmt.Sprintf("received raft log data %s", string(data)))
	// TODO : handle
	return nil
}

func (m *mutexBackend) Snapshot() (raft.FSMSnapshot, error) {
	// TODO : snapshot
	return &mutexSnapshot{}, nil
}

func (m *mutexBackend) Restore(r io.ReadCloser) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.lg.Info(fmt.Sprintf("restoring mutex state from snapshot %s", string(b)))
	// TODO : handle restores
	return nil
}

type mutexSnapshot struct {
	_ mutexBackend
}

var _ raft.FSMSnapshot = (*mutexSnapshot)(nil)

func (m *mutexSnapshot) Persist(sink raft.SnapshotSink) error {
	return nil
}

func (m *mutexSnapshot) Release() {

}

type backend struct {
	embeddedapi.UnsafeDistributedMutexServer

	raft *raft.Raft
	m    *mutexBackend
}

var _ embeddedapi.DistributedMutexServer = (*backend)(nil)

func (b *backend) Lock(*embeddedapi.LockRequest, embeddedapi.DistributedMutex_LockServer) error {
	// TODO call the relevant raft stuff

	f := b.raft.Apply([]byte("lock"), time.Second)
	if err := f.Error(); err != nil {
		return err
	}

	return nil
}

func (b *backend) Unlock(context.Context, *embeddedapi.LockRequest) (*emptypb.Empty, error) {
	// TODO : call the relevant raft stuff

	f := b.raft.Apply([]byte("unlock"), time.Second)
	if err := f.Error(); err != nil {
		return nil, err
	}

	return nil, nil
}

var _ node.RaftService = (*backend)(nil)

func (b *backend) ServiceDesc() *grpc.ServiceDesc {
	return &embeddedapi.DistributedMutex_ServiceDesc
}

func (b *backend) FSM() raft.FSM {
	return b.m
}

func (b *backend) Register(s *grpc.Server) {
	embeddedapi.RegisterDistributedMutexServer(s, b)
}

func (b *backend) Initialize(
	raft *raft.Raft,
) {
	b.raft = raft
}

func NewEmbeddedBackend() *backend {
	m := &mutexBackend{}
	return &backend{
		m:    m,
		raft: nil,
	}
}
