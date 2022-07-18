module github.com/froghub-io/filecoin-sealer-recover

go 1.15

replace github.com/filecoin-project/filecoin-ffi => ./extern/lotus/extern/filecoin-ffi

replace github.com/filecoin-project/lotus => ./extern/lotus

require (
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-state-types v0.1.10
	github.com/filecoin-project/lotus v1.16.1
	github.com/filecoin-project/specs-storage v0.4.1
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-log/v2 v2.5.1
	github.com/mitchellh/go-homedir v1.1.0
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/xerrors v0.0.0-20220411194840-2f41105eb62f
)
