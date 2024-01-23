package ds

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type Group struct {
	ID          uint   `gorm:"primaryKey;AUTO_INCREMENT"`
	Title       string `gorm:"type:varchar(255);unique;not null"`
	Course      string `gorm:"type:text"`
	Schedule    string `gorm:"type:text"`
	Location    string `gorm:"type:varchar(255);not null"`
	Status      string `gorm:"type:varchar(50);not null"`
	CoachName   string `gorm:"type:varchar(200)"`
	CoachPhone  string `gorm:"type:varchar(35)"`
	CoachEmail  string `gorm:"type:varchar(100)"`
	Capacity    uint   `gorm:"type:integer"`
	Enrolled    uint   `gorm:"type:integer"`
	Description string `gorm:"type:text"`
	ImageName   string
}

type Enrollment struct {
	ID             uint           `gorm:"primaryKey;AUTO_INCREMENT"`
	ModeratorRefer uuid.UUID      `gorm:"type:uuid"`
	UserRefer      uuid.UUID      `gorm:"type:uuid;not null"`
	Status         string         `gorm:"type:varchar(50);not null"`
	DateCreated    datatypes.Date `gorm:"not null" swaggertype:"primitive,string"`
	DateProcessed  datatypes.Date `swaggertype:"primitive,string"`
	DateFinished   datatypes.Date `swaggertype:"primitive,string"`
	Moderator      User           `gorm:"foreignKey:ModeratorRefer;references:UUID"`
	User           User           `gorm:"foreignKey:UserRefer;references:UUID;not null"`
}

type EnrollmentToGroup struct {
	ID              uint       `gorm:"primaryKey;AUTO_INCREMENT"`
	EnrollmentRefer int        `gorm:"not null"`
	GroupRefer      int        `gorm:"not null"`
	Enrollment      Enrollment `gorm:"foreignKey:EnrollmentRefer"`
	Group           Group      `gorm:"foreignKey:GroupRefer"`
	Status          string     `gorm:"type:varchar(50);not null"`
}
