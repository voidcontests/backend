package main

import (
	"github.com/voidcontests/backend/internal/config"
	"github.com/voidcontests/backend/internal/pkg/app"
)

func main() {
	c := config.MustLoad()
	a := app.New(c)

	a.Run()
}
