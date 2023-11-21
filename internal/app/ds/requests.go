package ds

type GetCoursesRequestBody struct {
	Location string
	Status   string
}

type GetEnrollmentsRequestBody struct {
	Status string
}

type EnrollCourseRequestBody struct {
	UserName   string
	StartDate  string
	EndDate    string
	CourseName string
}

type ChangeEnrollmentStatusRequestBody struct {
	ID       int
	Status   string
	UserName string
}

type DeleteEnrollmentToCourseRequestBody struct {
	EnrollmentID int
	CourseID     int
}
