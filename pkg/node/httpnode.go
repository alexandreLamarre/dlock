package node

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/alexandreLamarre/dlock/pkg/logger"
	"github.com/hashicorp/raft"
	"github.com/spf13/cobra"
)

func BuildNodeCmd() *cobra.Command {
	var raftAddr string
	var httpAddr string
	var joinAddr string
	var nodeId string
	var raftDir string
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Run a dlock raft node",
		RunE: func(cmd *cobra.Command, args []string) error {
			lg := logger.New().WithGroup("node")

			n := NewNodeSpec(raftAddr, httpAddr, joinAddr, nodeId, raftDir)
			node, err := NewNode(n, lg)
			if err != nil {
				return err
			}
			defer node.Stop()
			terminate := make(chan os.Signal, 1)
			signal.Notify(terminate, os.Interrupt)

			lg.With(
				"raftAddr", n.raftAddr,
				"httpAddr", n.httpAddr,
				"joinAddr", n.joinAddr,
				"nodeId", n.nodeId,
				"raftDir", n.raftDir,
			).Info("node up running successfully")
			<-terminate
			log.Println("node exiting")
			return nil
		},
	}
	cmd.Flags().StringVarP(&raftAddr, "raft-addr", "r", "127.0.0.1:5006", "advertise address for the RAFT backend")
	cmd.Flags().StringVarP(&httpAddr, "http-addr", "a", "127.0.0.1:5007", "address for the HTTP API")
	cmd.Flags().StringVarP(&joinAddr, "join-addr", "j", "", "address of an existing node to join")
	cmd.Flags().StringVarP(&nodeId, "node-id", "n", "", "unique node identifier")
	cmd.Flags().StringVarP(&raftDir, "raft-dir", "d", "", "raft storage directory, defaults to ./default.raft.<nodeId>")
	return cmd
}

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
)

type command struct {
	Op    string `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type NodeSpec struct {
	raftAddr string
	httpAddr string
	joinAddr string
	nodeId   string
	raftDir  string
}

func NewNodeSpec(
	raftAddr,
	httpAddr,
	joinAddr,
	nodeId,
	raftDir string,
) *NodeSpec {
	spec := &NodeSpec{
		raftAddr: raftAddr,
		httpAddr: httpAddr,
		joinAddr: joinAddr,
		nodeId:   nodeId,
		raftDir:  raftDir,
	}
	// TODO : in container land this check is terrible
	if spec.nodeId == "" {
		spec.nodeId = spec.raftAddr
	}
	if spec.raftDir == "" {
		spec.raftDir = fmt.Sprintf("default.raft.%s", spec.nodeId)
	}
	return spec
}
func (s *NodeSpec) Validate() error {
	if s.raftAddr == "" {
		return errors.New("raftAddr is required")
	}
	if s.httpAddr == "" {
		return errors.New("httpAddr is required")
	}
	return nil
}

type Node struct {
	spec *NodeSpec
	lg   *slog.Logger

	store Store
	ln    net.Listener
}

func NewNode(
	spec *NodeSpec,
	lg *slog.Logger,
) (*Node, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if spec.nodeId == "" {
		spec.nodeId = spec.raftAddr
	}
	lg = lg.With("nodeId", spec.nodeId)
	if err := os.MkdirAll(spec.raftDir, 0700); err != nil {
		lg.Error(fmt.Sprintf("failed to create path for Raft storage: %s", err.Error()))
		return nil, err
	}
	s := NewStore(lg.WithGroup("store"), spec.raftDir, spec.raftAddr)
	n := &Node{
		spec:  spec,
		lg:    lg,
		store: s,
	}

	// 1. open the store (starts serving the raft backend /config)
	if err := n.store.Open(spec.joinAddr == "", spec.nodeId); err != nil {
		lg.With(logger.Err(err)).Error("failed to open store")
		panic(err)
	}

	// 2. start http api
	if err := n.Start(); err != nil {
		lg.With(logger.Err(err)).Error("failed to start HTTP server")
		panic(err)
	}

	// 3. join happens last
	if spec.joinAddr == "" {
		lg.Warn("no join address specified, starting as single-node cluster")
	} else {
		if err := n.join(); err != nil {
			lg.With(logger.Err(err)).Error("failed to join single-node cluster")
			panic(err)
		}
	}
	return n, nil
}

func (n *Node) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/join", n.handleJoin)
	mux.HandleFunc("/key/*", n.handleKey)

	server := http.Server{
		Addr:    n.spec.httpAddr,
		Handler: mux,
	}

	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return err
	}
	n.ln = ln

	go func() {
		err := server.Serve(n.ln)
		if err != nil {
			log.Fatalf("HTTP serve: %s", err)
		}
	}()
	return nil
}

func (n *Node) Stop() error {
	return n.ln.Close()
}

func (n *Node) handleKey(w http.ResponseWriter, r *http.Request) {
	getKey := func() string {
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 3 {
			return ""
		}
		return parts[2]
	}

	switch r.Method {
	case "GET":
		k := getKey()
		if k == "" {
			w.WriteHeader(http.StatusBadRequest)
		}
		v, err := n.store.Get(k)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		b, err := json.Marshal(map[string]string{k: v})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		io.WriteString(w, string(b))

	case "POST":
		// Read the value from the POST body.
		m := map[string]string{}
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		for k, v := range m {
			if err := n.store.Set(k, v); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

	case "DELETE":
		k := getKey()
		if k == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := n.store.Delete(k); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		n.store.Delete(k)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (n *Node) join() error {
	b, err := json.Marshal(map[string]string{"addr": n.spec.raftAddr, "id": n.spec.nodeId})
	if err != nil {
		return err
	}
	resp, err := http.Post(fmt.Sprintf("http://%s/join", n.spec.joinAddr), "application-type/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (n *Node) handleJoin(w http.ResponseWriter, r *http.Request) {
	n.lg.Info("received join http request")
	m := map[string]string{}
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(m) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	remoteAddr, ok := m["addr"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	nodeID, ok := m["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := n.store.Join(nodeID, remoteAddr); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type Store interface {
	// Get returns the value for the given key.
	Get(key string) (string, error)

	// Set sets the value for the given key, via distributed consensus.
	Set(key, value string) error

	// Delete removes the given key, via distributed consensus.
	Delete(key string) error

	// Join joins the node, identitifed by nodeID and reachable at addr, to the cluster.
	Join(nodeID string, addr string) error
	// Open opens the store. If enableSingle is set, and there are no existing peers,
	// then this node becomes the first node, and therefore leader, of the cluster.
	// localID should be the server identifier for this node.
	Open(enableSingle bool, localID string) error
}

type storeImpl struct {
	raftDir  string
	raftBind string

	mu sync.RWMutex

	lg   *slog.Logger
	m    map[string]string
	raft *raft.Raft
}

func NewStore(
	lg *slog.Logger,
	raftDir, raftBind string,
) Store {
	return &storeImpl{
		m:        make(map[string]string),
		lg:       lg,
		raftDir:  raftDir,
		raftBind: raftBind,
	}
}

func (s *storeImpl) Open(enableSingle bool, localID string) error {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(localID)

	addr, err := net.ResolveTCPAddr("tcp", s.raftBind)
	if err != nil {
		s.lg.With(logger.Err(err)).Error("failed to resolve TCP address")
		panic(err)
	}
	transport, err := raft.NewTCPTransport(s.raftBind, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return err
	}

	snapshots, err := raft.NewFileSnapshotStore(s.raftDir, 2, os.Stderr)
	if err != nil {
		return fmt.Errorf("failed to create snapshot store: %s for raft", err)
	}

	var logStore raft.LogStore
	var stableStore raft.StableStore
	if true {
		logStore = raft.NewInmemStore()
		stableStore = raft.NewInmemStore()
	} else {
		// TODO
		fmt.Println("TODO : remote backend for raft")
	}
	ra, err := raft.NewRaft(config, (*fsm)(s), logStore, stableStore, snapshots, transport)
	if err != nil {
		return fmt.Errorf("failed to create raft: %s", err)
	}
	s.raft = ra
	if enableSingle {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		ra.BootstrapCluster(configuration)
	}
	return nil
}

func (s *storeImpl) Get(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[key], nil
}

func (s *storeImpl) Set(key, value string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	c := &command{
		Op:    "set",
		Key:   key,
		Value: value,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	f := s.raft.Apply(b, raftTimeout)
	return f.Error()
}

func (s *storeImpl) Delete(key string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not leader")
	}

	c := &command{
		Op:  "delete",
		Key: key,
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	f := s.raft.Apply(b, raftTimeout)
	return f.Error()
}

func (s *storeImpl) Join(curNodeId string, addr string) error {
	s.lg.Info("store received join request for remote node %s at %s", curNodeId, addr)
	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		s.lg.Error("failed to get raft configuration: %s", err)
		return err
	}

	for _, server := range configFuture.Configuration().Servers {
		// if a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first
		if server.ID == raft.ServerID(curNodeId) || server.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if server.Address == raft.ServerAddress(addr) && server.ID == raft.ServerID(curNodeId) {
				s.lg.Warn(
					fmt.Sprintf(
						"node %s at %s already member of cluster, ignoring join request",
						curNodeId,
						addr,
					),
				)
				return nil
			}
			future := s.raft.RemoveServer(server.ID, 0, 0)
			if err := future.Error(); err != nil {
				s.lg.Error(
					fmt.Sprintf(
						"failed to remove existing node %s at %s from raft configuration: %s",
						curNodeId,
						addr,
						err,
					),
				)
				return err
			}
		}
	}
	f := s.raft.AddVoter(raft.ServerID(curNodeId), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		s.lg.Error("failed to add node %s at %s to raft: %s", curNodeId, addr, f.Error())
		return f.Error()
	}
	s.lg.Info("node %s at %s joined successfully", curNodeId, addr)
	return nil
}

type fsm storeImpl

var _ raft.FSM = (*fsm)(nil)

// Apply applies a Raft log entry to the key-value store.
func (f *fsm) Apply(l *raft.Log) interface{} {
	var c command
	if err := json.Unmarshal(l.Data, &c); err != nil {
		panic(fmt.Sprintf("failed to unmarshal command: %s", err))
	}

	switch c.Op {
	case "set":
		return f.applySet(c.Key, c.Value)
	case "delete":
		return f.applyDelete(c.Key)
	default:
		panic(fmt.Sprintf("unrecognized command op: %s", c.Op))
	}
}

// Snapshot returns a snapshot of the key-value store.
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Clone the map
	o := make(map[string]string, len(f.m))
	for k, v := range f.m {
		o[k] = v
	}
	return &fsmSnapshot{store: o}, nil
}

// Restore stores the key-value store to a previous state.

func (f *fsm) Restore(rc io.ReadCloser) error {
	o := make(map[string]string)
	if err := json.NewDecoder(rc).Decode(&o); err != nil {
		return err
	}

	// Set the state from the snapshot, no lock required according to
	// Hashicorp docs.
	f.m = o
	return nil
}

func (f *fsm) applySet(key, value string) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[key] = value
	return nil
}

func (f *fsm) applyDelete(key string) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.m, key)
	return nil
}

type fsmSnapshot struct {
	store map[string]string
}

func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		// Encode data.
		b, err := json.Marshal(f.store)
		if err != nil {
			return err
		}

		// Write data to sink.
		if _, err := sink.Write(b); err != nil {
			return err
		}

		// Close the sink.
		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
	}

	return err
}

func (f *fsmSnapshot) Release() {}
