package test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/alexandreLamarre/dlock/pkg/config/v1alpha1"
	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/alexandreLamarre/dlock/pkg/test/freeport"
	natsserver "github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gexec"
	goredislib "github.com/redis/go-redis/v9"
	"github.com/stvp/tempredis"
	etcdserver "go.etcd.io/etcd/server/v3/embed"
)

func (e *Environment) Start() error {
	if e.ctx == nil {
		e.ctx = context.Background()
	}
	if e.Logger == nil {
		e.Logger = logger.New()
	}
	path, err := os.MkdirTemp("/tmp", "dlock-test")
	if err != nil {
		return err
	}
	e.tempDir = path
	return nil

}

type Environment struct {
	Logger *slog.Logger

	tempDir      string
	ctx          context.Context
	cancel       context.CancelFunc
	TestBin      string
	embeddedJS   *natsserver.Server
	embeddedEtcd *etcdserver.Etcd

	shutdownHooks []func()
}

func (e *Environment) Stop(cause ...string) error {
	if len(cause) > 0 {
		e.Logger.With(
			"cause", cause[0],
		).Info("Stopping test environment")
	} else {
		e.Logger.Info("Stopping test environment")
	}

	if e.cancel != nil {
		e.cancel()
		var wg sync.WaitGroup
		for _, h := range e.shutdownHooks {
			wg.Add(1)
			h := h
			go func() {
				defer wg.Done()
				h()
			}()
		}
		wg.Wait()
	}
	// if e.mockCtrl != nil {
	// 	e.mockCtrl.Finish()
	// }
	if e.tempDir != "" {
		os.RemoveAll(e.tempDir)
	}
	return nil
}

func (e *Environment) addShutdownHook(fn func()) {
	e.shutdownHooks = append(e.shutdownHooks, fn)
}

type Session interface {
	G() (*gexec.Session, bool)
	Wait() error
}

type sessionWrapper struct {
	g   *gexec.Session
	cmd *exec.Cmd
}

func (s *sessionWrapper) G() (*gexec.Session, bool) {
	if s.g != nil {
		return s.g, true
	}
	return nil, false
}

func (s *sessionWrapper) Wait() error {
	if s == nil {
		return nil
	}
	if s.g != nil {
		ws := s.g.Wait()
		if ws.ExitCode() != 0 {
			return errors.New(string(ws.Err.Contents()))
		}
		return nil
	}
	return s.cmd.Wait()
}

func StartCmd(cmd *exec.Cmd) (Session, error) {
	session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	if err != nil {
		return nil, err
	}
	return &sessionWrapper{
		g:   session,
		cmd: cmd,
	}, nil

}

func (e *Environment) StartRedis() ([]*goredislib.Options, error) {
	server, err := tempredis.Start(
		tempredis.Config{},
		tempredis.WithWriter(ginkgo.GinkgoWriter),
	)
	if err != nil {
		return nil, err
	}
	e.addShutdownHook(func() {
		server.Term()
	})
	e.Logger.Info("Redis server started", "socket", server.Socket())
	return []*goredislib.Options{
		{
			Network: "unix",
			Addr:    server.Socket(),
		}}, nil
}

// FIXME: set the ports to freeports
func (e *Environment) StartEtcd() (*v1alpha1.EtcdClientSpec, error) {
	conf := etcdserver.NewConfig()
	if err := conf.Validate(); err != nil {
		panic(err)
	}

	conf.Dir = path.Join(e.tempDir, "etcd-data")
	conf.WalDir = path.Join(e.tempDir, "etcd-wal")
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	server, err := etcdserver.StartEtcd(conf)
	if err != nil {
		return nil, err
	}
	e.embeddedEtcd = server
	e.addShutdownHook(func() {
		server.Close()
	})
	e.Logger.Info("Etcd server started", "endpoints", etcdserver.DefaultAdvertiseClientURLs)
	return &v1alpha1.EtcdClientSpec{
		Endpoints: []string{etcdserver.DefaultAdvertiseClientURLs},
	}, nil
}

func (e *Environment) StartJetstream() (*v1alpha1.JetstreamClientSpec, error) {
	ports := freeport.GetFreePorts(1)

	opts := natstest.DefaultTestOptions
	opts.Port = ports[0]
	opts.StoreDir = e.tempDir

	server := natstest.RunServer(&opts)
	e.embeddedJS = server
	e.addShutdownHook(func() {
		server.Shutdown()
	})
	e.embeddedJS.EnableJetStream(nil)
	if !e.embeddedJS.ReadyForConnections(2 * time.Second) {
		return nil, errors.New("starting nats server: timeout")
	}
	e.Logger.Info("Jetstream server started", "port", ports[0])

	sUrl := fmt.Sprintf("nats://127.0.0.1:%d", ports[0])
	return &v1alpha1.JetstreamClientSpec{
		Endpoint: sUrl,
	}, nil
}
