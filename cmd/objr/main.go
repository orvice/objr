package main

import (
	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"
	"github.com/orvice/objr/internal/apis"
	"github.com/orvice/objr/internal/conf"
	"github.com/orvice/objr/internal/object"
)

func main() {

	app := core.New(&app.Config{
		Service: "objr",
		Router:  apis.Router,
		Config:  conf.Conf,
		InitFunc: []func() error{
			object.Init,
		},
	})
	app.Run()
}
