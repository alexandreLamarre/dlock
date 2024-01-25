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

	os.Unsetenv("NATS_SERVER_URL")
	os.Unsetenv("NKEY_SEED_FILENAME")

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
	if e.embeddedJS != nil {
		e.embeddedJS.Shutdown()
	}
	if e.embeddedEtcd != nil {
		e.embeddedEtcd.Close()
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

// FIXME: set the ports to freeports
func (e *Environment) StartEtcd() (*v1alpha1.EtcdStorageSpec, error) {
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
	return &v1alpha1.EtcdStorageSpec{
		Endpoints: []string{etcdserver.DefaultAdvertiseClientURLs},
	}, nil
}

func (e *Environment) StartJetstream() (*v1alpha1.JetStreamStorageSpec, error) {
	ports := freeport.GetFreePorts(1)

	opts := natstest.DefaultTestOptions
	opts.Port = ports[0]
	opts.StoreDir = e.tempDir

	e.embeddedJS = natstest.RunServer(&opts)
	e.embeddedJS.EnableJetStream(nil)
	if !e.embeddedJS.ReadyForConnections(2 * time.Second) {
		return nil, errors.New("starting nats server: timeout")
	}

	sUrl := fmt.Sprintf("nats://127.0.0.1:%d", ports[0])
	return &v1alpha1.JetStreamStorageSpec{
		Endpoint: sUrl,
	}, nil
}

func StartEtcd() *v1alpha1.EtcdStorageSpec {
	return &v1alpha1.EtcdStorageSpec{}
}
