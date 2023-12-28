package ds

type CourseStatus int
type EnrollmentStatus int

const (
	Draft EnrollmentStatus = iota
	Formed
	Completed
	Rejected
	Deleted
)

const (
	Active CourseStatus = iota
	Inactive
)
