package types

import "time"

type Project struct {
	ID          string      `bson:"_id" json:"_id"`
	Name        string      `bson:"projectName" json:"projectName" validate:"required"`
	Description *string     `bson:"description,omitempty" json:"description"`
	CreatedAt   time.Time   `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time   `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	Archived    bool        `bson:"archived" json:"archived"`
	Type        ProjectType `bson:"projectType" json:"projectType" validate:"required"`
}

type ProjectType string

const (
	Work     ProjectType = "WORK"
	Holidays ProjectType = "HOLIDAYS"
	Sickness ProjectType = "SICKNESS"
	Absence  ProjectType = "ABSENCE"
)

type PlanningEntry struct {
	ID                      string     `bson:"_id" json:"_id"`
	ProjectID               string     `bson:"projectId" json:"projectId" validate:"required"`
	CreatedAt               time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt               *time.Time `bson:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	Start                   string     `bson:"start" json:"start" validate:"required"`
	End                     string     `bson:"end" json:"end" validate:"required"`
	EmployeeIDs             []string   `bson:"employeeIds, omitempty" json:"employeeIds"`
	AllowMultipleAssignment bool       `bson:"multipleAssignment" json:"multipleAssignment"`
	Title                   string     `bson:"title" json:"title" validate:"required"`
	Description             *string    `bson:"description,omitempty" json:"description"`
	Comments                []Comment  `bson:"comments" json:"comments"`
}

type PlanningAssignment struct {
	ID         string    `bson:"_id" json:"_id"`
	EntryID    string    `bson:"entryId" json:"entryId"`
	EmployeeID string    `bson:"employeeId" json:"employeeId"`
	CreatedAt  time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time `bson:"updatedAt,omitempty" json:"updatedAt"`
	SendDate   time.Time `bson:"sendDate,omitempty" json:"sendDate"`
	Cancelled  bool      `bson:"cancelled" json:"cancelled"`
}

type (
	PlanningCycle struct {
		ProjectID               string                `json:"projectId" validate:"required"`
		Start                   string                `json:"start" validate:"required"`
		End                     string                `json:"end" validate:"required"`
		EmployeeIDs             []string              `json:"employeeIds"`
		AllowMultipleAssignment bool                  `json:"multipleAssignment"`
		Title                   string                `json:"title" validate:"required"`
		Description             *string               `json:"description"`
		RotationFrequency       uint16                `json:"rotationFrequency" validate:"required,min=1"`
		RotationFrequencyType   RotationFrequencyType `json:"rotationFrequencyType" validate:"required"`
		Shifts                  []Shift               `json:"shifts" validate:"required,min=1"`
		IncludeSaturday         bool                  `json:"includeSaturday"`
		IncludeSunday           bool                  `json:"includeSunday"`
	}
	Shift struct {
		StartHour   int `json:"startHour" validate:"required,min=0,max=23"`
		StartMinute int `json:"startMinute" validate:"required,min=0,max=59"`
		EndHour     int `json:"endHour" validate:"required,min=0,max=23"`
		EndMinute   int `json:"endMinute" validate:"required,min=0,max=59"`
	}
	RotationFrequencyType = string
)

const (
	Days  RotationFrequencyType = "DAYS"
	Weeks RotationFrequencyType = "WEEKS"
)

type PlanningAssignmentDetail struct {
	PlanningAssignment `bson:",inline"`
	Entry              *PlanningEntry `bson:"entry" json:"entry,omitempty"`
	Project            *Project       `bson:"project" json:"project,omitempty"`
}

func (entry PlanningAssignmentDetail) GetID() string {
	return entry.ID
}

func (entry PlanningAssignment) GetID() string {
	return entry.ID
}

func (entry *PlanningAssignment) SetID(id string) {
	entry.ID = id
}

func (entry Project) GetID() string {
	return entry.ID
}

func (entry *Project) SetID(id string) {
	entry.ID = id
}

func (entry PlanningEntry) GetID() string {
	return entry.ID
}

func (entry *PlanningEntry) SetID(id string) {
	entry.ID = id
}
