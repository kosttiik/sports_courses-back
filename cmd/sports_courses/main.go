package main

import (
	"log"

	"sports_courses/internal/api"
)

func main() {
	log.Println("Application started!")
	api.StartServer()
	log.Println("Application terminated.")
}
