package main

import (
	"fmt"
	"os"

	"github.com/JinFuuMugen/GophKeeper/internal/cliapp"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

func main() {
	app := cliapp.New(cliapp.BuildInfo{
		Version: buildVersion,
		Date:    buildDate,
		Commit:  buildCommit,
	})

	if err := app.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
