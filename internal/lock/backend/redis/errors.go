package redis

import (
	"errors"
	"fmt"
)

var ErrExtendFailed = errors.New("redsync: failed to extend lock")

// A RedisError is an error communicating with one of the Redis nodes.
type RedisError struct {
	Node int
	Err  error
}

func (err RedisError) Error() string {
	return fmt.Sprintf("node %d: %v", err.Node, err.Err)
}

// ErrNodeTaken is the error resulting if the lock is already taken in one of
// the cluster's nodes
type ErrNodeTaken struct {
	Node int
}

func (err ErrNodeTaken) Error() string {
	return fmt.Sprintf("node #%d: lock already taken", err.Node)
}

var ErrTaken = errors.New("lock already taken")
