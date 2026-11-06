package main

import (
	"github.com/keyCat/srcds-manager/cmd"
	"github.com/keyCat/srcds-manager/utils"
)

var version = "1.1.2"

func main() {
	cmd.SetVersion(version)
	utils.FatalIfCommandIsNotAvailable("taskset")
	utils.FatalIfCommandIsNotAvailable("nice")
	utils.FatalIfCommandIsNotAvailable("screen")
	cmd.Execute()
}
