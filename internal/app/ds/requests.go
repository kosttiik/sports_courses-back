package ds

type EnrollRequestBody struct {
	Groups []string
	Status string
}

type EditEnrollmentRequestBody struct {
	EnrollmentID int `json:"enrollmentID"`
}

type SetEnrollmentGroupsRequestBody struct {
	EnrollmentID int
	Groups       []string
}

type ChangeEnrollmentStatusRequestBody struct {
	EnrollmentID int
	Status       string
}

type ChangeEnrollmentToGroupAvailabilityRequestBody struct {
	EnrollmentID int
	Availability string
}

type DeleteEnrollmentToGroupRequestBody struct {
	EnrollmentID int
	GroupID      int
}
