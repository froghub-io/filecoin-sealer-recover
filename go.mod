module github.com/froghub-io/filecoin-sealer-recover

go 1.15

replace github.com/filecoin-project/filecoin-ffi => ./extern/lotus/extern/filecoin-ffi

replace github.com/filecoin-project/lotus => ./extern/lotus

require (
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-state-types v0.1.3
	github.com/filecoin-project/lotus v1.15.0
	github.com/filecoin-project/specs-storage v0.2.0
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-log/v2 v2.4.0
	github.com/mitchellh/go-homedir v1.1.0
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)
