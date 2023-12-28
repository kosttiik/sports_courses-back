package ds

type RegionStatus int
type FlightStatus int

const (
	Draft FlightStatus = iota
	Formed
	Completed
	Rejected
	Deleted
)

const (
	Active RegionStatus = iota
	Inactive
)
