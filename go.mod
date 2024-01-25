module github.com/alexandreLamarre/dlock

go 1.21.3

replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.8

replace google.golang.org/grpc => google.golang.org/grpc v1.32.0

require (
	github.com/google/uuid v1.6.0
	github.com/jwalton/go-supportscolor v1.2.0
	github.com/kralicky/gpkg v0.0.0-20240119195700-64f32830b14f
	github.com/lestrrat-go/backoff/v2 v2.0.8
	github.com/nats-io/nats.go v1.32.0
	github.com/onsi/ginkgo/v2 v2.15.0
	github.com/onsi/gomega v1.31.1
	github.com/samber/lo v1.39.0
	github.com/samber/slog-multi v1.0.2
	github.com/samber/slog-sampling v1.4.0
	github.com/spf13/cobra v1.8.0
	github.com/ttacon/chalk v0.0.0-20160626202418-22c06c80ed31
	go.etcd.io/etcd/client/v3 v3.5.9
	golang.org/x/sync v0.5.0
	google.golang.org/grpc v1.58.3
	google.golang.org/protobuf v1.31.0
)

require (
	github.com/bluele/gcache v0.0.2 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/cornelk/hashmap v1.0.8 // indirect
	github.com/go-logr/logr v1.3.0 // indirect
	github.com/go-task/slim-sprig v0.0.0-20230315185526-52ccab3ef572 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/lestrrat-go/option v1.0.0 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/samber/slog-common v0.14.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.etcd.io/etcd/api/v3 v3.5.10 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.9 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.18.1 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/exp v0.0.0-20230522175609-2e198f4a06a1 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/term v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/tools v0.16.1 // indirect
	google.golang.org/genproto v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230711160842-782d3b101e98 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
