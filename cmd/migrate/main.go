package main

import (
	"sports_courses/internal/app/ds"
	"sports_courses/internal/app/dsn"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()
	db, err := gorm.Open(postgres.Open(dsn.FromEnv()), &gorm.Config{})
	if err != nil {
		panic("failed to connect database! :(")
	}

	MigrateSchema(db)
}

func MigrateSchema(db *gorm.DB) {
	err := db.AutoMigrate(&ds.User{})
	err = db.AutoMigrate(&ds.Group{})
	err = db.AutoMigrate(&ds.Enrollment{})
	err = db.AutoMigrate(&ds.EnrollmentToGroup{})

	if err != nil {
		panic(err)
	}
}
