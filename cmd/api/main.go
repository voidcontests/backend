package main

import (
	"github.com/voidcontests/api/internal/config"
	"github.com/voidcontests/api/internal/pkg/app"
)

func main() {
	c := config.MustLoad()
	a := app.New(c)

	a.Run()
}
