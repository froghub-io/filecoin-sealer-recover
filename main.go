package main

import (
	"github.com/froghub-io/filecoin-sealer-recover/export"
	"github.com/froghub-io/filecoin-sealer-recover/recovery"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	app := &cli.App{
		Name:    "sealer-recovery",
		Usage:   "Filecoin sealer recovery",
		Version: BuildVersion,
		Commands: []*cli.Command{
			recovery.RecoverCmd,
			export.ExportCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Warnf("%+v", err)
		return
	}
}

// BuildVersion is the local build version
const BuildVersion = "1.0.0"
