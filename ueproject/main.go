package main

import (
	"eptablegenerator/table/config"
	"eptablegenerator/ueproject/gen"
	"os"
)

func main() {
	var c config.Config

	if len(os.Args) > 1 {
		c = *config.LoadConfig(os.Args[1])
	} else {
		c = *config.NewConfig()
	}

	c.SourceDir = "../../SProject/XLSX"

	if err := gen.GenerateUE(&c); err != nil {
		panic(err)
	}
}
