package ds

type GroupStatus int
type EnrollmentStatus int

const (
	Draft EnrollmentStatus = iota
	Formed
	Completed
	Rejected
	Deleted
)

const (
	Active GroupStatus = iota
	Inactive
)
