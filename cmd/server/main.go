package main

import (
	"github.com/cascadecontests/backend/internal/config"
	"github.com/cascadecontests/backend/internal/pkg/app"
)

func main() {
	c := config.MustLoad()
	a := app.New(c)

	a.Run()
}
