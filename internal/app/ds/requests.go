package ds

type EnrollCourseRequestBody struct {
	StartDate  string
	EndDate    string
	CourseName string
}

type ChangeEnrollmentStatusRequestBody struct {
	ID     int
	Status string
}

type DeleteEnrollmentToCourseRequestBody struct {
	EnrollmentID int
	CourseID     int
}
