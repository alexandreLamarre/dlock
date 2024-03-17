package node

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"

	"github.com/Jille/raft-grpc-leader-rpc/leaderhealth"
	transport "github.com/Jille/raft-grpc-transport"
	"github.com/Jille/raftadmin"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/hashicorp/raft"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

// TODO : this is too janky to actually use for prod

type RaftService interface {
	ServiceDesc() *grpc.ServiceDesc
	FSM() raft.FSM
	Register(*grpc.Server)
	Initialize(*raft.Raft)
}

type errLogRedirection struct {
	lg *slog.Logger
}

var _ io.Writer = (*errLogRedirection)(nil)

func (e *errLogRedirection) Write(p []byte) (n int, err error) {
	e.lg.Warn(string(p))
	return len(p), nil
}

type grpcRaft struct {
	ctx  context.Context
	spec RaftSpec

	lg *slog.Logger

	raftSvc RaftService
	tm      *transport.Manager
	raft    *raft.Raft
}

type RaftSpec struct {
	RaftDir       string
	NodeId        string
	NodeAddr      string
	JoinAddrs     []string
	SingleCluster bool
}

func NewGrpcRaft(
	ctx context.Context,
	spec RaftSpec,
	raftService RaftService,
) *grpcRaft {
	return &grpcRaft{
		ctx:     ctx,
		lg:      logger.New().WithGroup("grpc-raft"),
		raftSvc: raftService,
		spec:    spec,
	}
}

func (g *grpcRaft) Start(ctx context.Context, fsm raft.FSM) error {
	raftInstance, tm, err := g.Open()
	if err != nil {
		return err
	}
	g.raftSvc.Initialize(raftInstance)
	g.raft = raftInstance
	g.tm = tm
	return nil
}

func (g *grpcRaft) ListenAndServe() error {
	sock, err := net.Listen("tcp", g.spec.NodeAddr)
	if err != nil {
		g.lg.With(logger.Err(err)).Error("failed to listen on address")
		return err
	}

	if err := g.Start(context.Background(), g.raftSvc.FSM()); err != nil {
		return err
	}

	s := grpc.NewServer()
	g.raftSvc.Register(s)
	g.tm.Register(s)
	leaderhealth.Setup(g.raft, s, []string{g.raftSvc.ServiceDesc().ServiceName})
	raftadmin.Register(s, g.raft)
	reflection.Register(s)

	err = s.Serve(sock)
	if err != nil {
		g.lg.With(logger.Err(err)).Error("failed to serve grpc server")
	}
	return err
}

func (g *grpcRaft) Register(grpcServer *grpc.Server) {
	grpcServer.RegisterService(g.raftSvc.ServiceDesc(), g.raftSvc)
	g.tm.Register(grpcServer)
	leaderhealth.Setup(g.raft, grpcServer, []string{
		g.raftSvc.ServiceDesc().ServiceName,
	})
	raftadmin.Register(grpcServer, g.raft)
}

func newLogStore() (raft.LogStore, error) {
	return raft.NewInmemStore(), nil
}

func newStableStore() (raft.StableStore, error) {
	return raft.NewInmemStore(), nil
}

func (g *grpcRaft) Open() (*raft.Raft, *transport.Manager, error) {
	c := raft.DefaultConfig()
	c.LocalID = raft.ServerID(g.spec.NodeId)
	logsStore, err := newLogStore()
	if err != nil {
		g.lg.With(logger.Err(err)).Error("failed to create new log store")
		return nil, nil, err
	}
	stableStore, err := newStableStore()
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(g.spec.RaftDir, 0700); err != nil {
		g.lg.With(logger.Err(err)).Error("failed to create snapshot directory")
		return nil, nil, err
	}

	fss, err := raft.NewFileSnapshotStore(g.spec.RaftDir, 3, &errLogRedirection{lg: g.lg.WithGroup("grpc-snapshot")})
	if err != nil {
		g.lg.With(logger.Err(err)).Error("failed to create new file snapshot store")
		return nil, nil, fmt.Errorf(`raft.NewFileSnapshotStore(%q, ...): %v`, g.spec.RaftDir, err)
	}

	tm := transport.New(raft.ServerAddress(g.spec.NodeAddr), []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	})

	raftInstance, err := raft.NewRaft(c, g.raftSvc.FSM(), logsStore, stableStore, fss, tm.Transport())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize new raft instance %s", err)
	}

	if len(g.spec.JoinAddrs) == 0 {
		g.lg.Info("bootstrapping as single cluster")
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      c.LocalID,
					Address: raft.ServerAddress(g.spec.NodeAddr),
				},
			},
		}
		f := raftInstance.BootstrapCluster(configuration)
		if f.Error() != nil {
			g.lg.Warn("failed to bootstrap cluster", "error", f.Error())
		}
	}
	return raftInstance, tm, nil
}
