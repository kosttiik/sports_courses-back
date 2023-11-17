package main

import (
	"log"

	"sports_courses/internal/pkg/app"
)

func main() {
	log.Println("Application started!")

	a := app.New()
	a.StartServer()

	log.Println("Application terminated.")
}
