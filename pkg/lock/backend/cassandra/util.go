package cassandra

import "github.com/gocql/gocql"

func NewCassandraSession(
	hosts ...string,
) (*gocql.Session, error) {
	cluster := gocql.NewCluster(hosts...)

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}
	return session, nil
}
