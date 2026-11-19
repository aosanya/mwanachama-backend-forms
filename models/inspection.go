package models

type InspectionResult string

const (
	InspectionComplies      InspectionResult = "complies"
	InspectionDoesNotComply InspectionResult = "does_not_comply"
	InspectionNeedsMore     InspectionResult = "needs_more"
)
