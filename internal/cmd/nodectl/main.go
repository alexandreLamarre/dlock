package main

import (
	"errors"
	"fmt"

	"github.com/alexandreLamarre/dlock/internal/embedded/server"
	"github.com/alexandreLamarre/dlock/pkg/node"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func main() {
	cmd := BuildNodeCmd()
	cmd.Execute()
}

func BuildNodeCmd() *cobra.Command {
	var raftDir string
	var nodeId string
	var nodeAddr string
	var nodePeers []string
	cmd := &cobra.Command{
		Use:   "nodectl",
		Short: "nodectl is a server for managing distributed locks",
		RunE: func(cmd *cobra.Command, args []string) error {

			if nodeId == "" {
				return errors.New("node id is required")
			}

			if nodeAddr == "" {
				return errors.New("node address is required")
			}

			if raftDir == "" {
				raftDir = "default.raft"
			}

			raftDir = fmt.Sprintf("%s.%s", raftDir, nodeId)

			node := node.NewGrpcRaft(
				cmd.Context(),
				node.RaftSpec{
					RaftDir:  raftDir,
					NodeId:   nodeId,
					NodeAddr: nodeAddr,
					// SingleCluster: true,
					JoinAddrs: nodePeers,
				},
				server.NewEmbeddedBackend(),
			)

			errC := lo.Async(node.ListenAndServe)

			select {
			case err := <-errC:
				return err
			case <-cmd.Context().Done():
				return cmd.Context().Err()
			}
		},
	}
	cmd.Flags().StringVarP(&raftDir, "raft-dir", "d", "", "raft data directory")
	cmd.Flags().StringVarP(&nodeId, "node-id", "i", "", "node id")
	cmd.Flags().StringVarP(&nodeAddr, "node-addr", "a", "", "node address")
	cmd.Flags().StringArrayVarP(&nodePeers, "node-peers", "j", []string{}, "addresses of node peers")
	return cmd
}
