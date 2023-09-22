package main

import (
	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"
	"github.com/orvice/objr/internal/apis"
	"github.com/orvice/objr/internal/object"
)

func main() {
	err := object.Init()
	if err != nil {
		panic(err)
	}

	app := core.New(&app.Config{
		Service: "echo",
		Router:  apis.Router,
	})
	app.Run()
}
