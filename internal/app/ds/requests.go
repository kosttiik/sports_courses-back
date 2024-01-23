package ds

type EnrollRequestBody struct {
	Groups []string
}

type EditEnrollmentRequestBody struct {
	EnrollmentID int    `json:"enrollmentID"`
	Status       string `json:"status"`
}

type SetEnrollmentGroupsRequestBody struct {
	EnrollmentID int
	Groups       []string
}

type ChangeEnrollmentStatusRequestBody struct {
	ID     int
	Status string
}

type ChangeEnrollmentToGroupStatusRequestBody struct {
	ID     int
	Status string
}

type DeleteEnrollmentToGroupRequestBody struct {
	EnrollmentID int
	GroupID      int
}
