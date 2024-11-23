package main

import (
	"github.com/cascadecontests/backend/internal/app"
	"github.com/cascadecontests/backend/internal/config"
)

func main() {
	c := config.MustLoad()
	a := app.New(c)

	a.Run()
}
