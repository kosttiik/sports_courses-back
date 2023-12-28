package ds

import "gorm.io/datatypes"

type EnrollRequestBody struct {
	StartDate string
	EndDate   string
	Courses   []string
}

type EditEnrollmentRequestBody struct {
	EnrollmentID int            `json:"enrollmentID"`
	StartDate    datatypes.Date `json:"startDate"`
	EndDate      datatypes.Date `json:"endDate"`
	Status       string         `json:"status"`
}

type SetEnrollmentCoursesRequestBody struct {
	EnrollmentID int
	Courses      []string
}

type ChangeEnrollmentStatusRequestBody struct {
	ID     int
	Status string
}

type DeleteEnrollmentToCourseRequestBody struct {
	EnrollmentID int
	CourseID     int
}
