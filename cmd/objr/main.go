package main

import (
	"context"

	"github.com/orvice/objr/internal/apis"
	"github.com/orvice/objr/internal/object"
	"github.com/weeon/weeon"
)

func main() {
	err := object.Init()
	if err != nil {
		panic(err)
	}

	app := weeon.New(context.Background(), &weeon.Config{
		HTTPRouter: apis.Router,
	})
	app.Run()
}
