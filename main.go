/*
 * Copyright (C) 2022 FrogHub
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
