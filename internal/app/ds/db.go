package ds

import (
	"gorm.io/datatypes"
)

type Course struct {
	ID          uint   `gorm:"primaryKey;AUTO_INCREMENT"`
	Title       string `gorm:"type:varchar(255);unique;not null"`
	Location    string `gorm:"type:varchar(255);not null"`
	Status      string `gorm:"type:varchar(50);not null"`
	CoachName   string `gorm:"type:varchar(200)"`
	CoachPhone  string `gorm:"type:varchar(35)"`
	CoachEmail  string `gorm:"type:varchar(100)"`
	Capacity    uint   `gorm:"type:integer"`
	Enrolled    uint   `gorm:"type:integer"`
	Description string `gorm:"type:text"`
	Image       string `gorm:"type:bytea"`
}

type User struct {
	ID   uint   `gorm:"primaryKey;AUTO_INCREMENT"`
	Name string `gorm:"type:varchar(100);not null"`
	Role string `gorm:"type:varchar(50);not null"`
}

type Enrollment struct {
	ID             uint `gorm:"primaryKey;AUTO_INCREMENT"`
	ModeratorRefer int
	UserRefer      int
	Status         string         `gorm:"type:varchar(50);not null"`
	DateCreated    datatypes.Date `gorm:"not null"`
	DateProcessed  datatypes.Date
	DateFinished   datatypes.Date
	Moderator      User           `gorm:"foreignKey:ModeratorRefer"`
	User           User           `gorm:"foreignKey:UserRefer"`
	StartDate      datatypes.Date `gorm:"not null"`
	EndDate        datatypes.Date `gorm:"not null"`
}

type EnrollmentToCourse struct {
	ID              uint       `gorm:"primaryKey;AUTO_INCREMENT"`
	EnrollmentRefer int        `gorm:"not null"`
	CourseRefer     int        `gorm:"not null"`
	Enrollment      Enrollment `gorm:"foreignKey:EnrollmentRefer"`
	Course          Course     `gorm:"foreignKey:CourseRefer"`
}
