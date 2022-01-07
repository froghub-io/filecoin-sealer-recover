package main

import (
	"github.com/froghub-io/filecoin-sealer-recover/export"
	"github.com/froghub-io/filecoin-sealer-recover/recovery"
	logging "github.com/ipfs/go-log/v2"
	"github.com/urfave/cli/v2"
	"os"
)

var log = logging.Logger("sealer-recover-main")

func main() {
	logging.SetLogLevel("*", "INFO")

	app := &cli.App{
		Name:    "sealer-recovery",
		Usage:   "Filecoin sealer recovery",
		Version: BuildVersion,
		Commands: []*cli.Command{
			recovery.RecoverCmd,
			export.ExportsCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		return
	}
}

// BuildVersion is the local build version
const BuildVersion = "1.1.0"
