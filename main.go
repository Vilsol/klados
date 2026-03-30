package main

import (
	"embed"

	"github.com/Vilsol/klados/cmd"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cmd.Execute(assets)
}
