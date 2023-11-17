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

	// Migrate the schema

	MigrateSchema(db)
}

func MigrateSchema(db *gorm.DB) {
	MigrateUser(db)
	MigrateCourse(db)
	MigrateEnrollment(db)
	MigrateEnrollmentToCourse(db)
}

func MigrateCourse(db *gorm.DB) {
	err := db.AutoMigrate(&ds.Course{})
	if err != nil {
		panic("cant migrate Course to db")
	}
}

func MigrateUser(db *gorm.DB) {
	err := db.AutoMigrate(&ds.User{})
	if err != nil {
		panic("cant migrate User to db")
	}
}

func MigrateEnrollment(db *gorm.DB) {
	err := db.AutoMigrate(&ds.Enrollment{})
	if err != nil {
		panic("cant migrate Enrollment to db")
	}
}

func MigrateEnrollmentToCourse(db *gorm.DB) {
	err := db.AutoMigrate(&ds.EnrollmentToCourse{})
	if err != nil {
		panic("cant migrate EnrollmentToCourse db")
	}
}
