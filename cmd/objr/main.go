package main

import (
	"github.com/orvice/objr/internal/apis"
	"github.com/orvice/objr/internal/object"
)

func main() {
	err := object.Init()
	if err != nil {
		panic(err)
	}
	apis.Router()
}
