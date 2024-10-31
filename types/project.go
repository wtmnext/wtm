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
	Start                   time.Time  `bson:"start" json:"start" validate:"required"`
	End                     time.Time  `bson:"end" json:"end" validate:"required"`
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
