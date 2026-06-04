package main

import (
	"log"

	"github.com/fastygo/app-suite/pkg/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
