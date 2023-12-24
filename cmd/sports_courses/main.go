package main

import (
	"log"

	"sports_courses/internal/pkg/app"
)

// @title Запись на спортивные курсы МГТУ им. Н. Э. Баумана
// @version 0.0-0

// @host 127.0.0.1:8080
// @schemes http
// @BasePath /

func main() {
	log.Println("Application started!")

	a := app.New()
	a.StartServer()

	log.Println("Application terminated.")
}
