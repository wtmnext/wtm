package services

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/nbittich/wtm/config"
	"github.com/nbittich/wtm/services/db"
	"github.com/nbittich/wtm/services/email"
	"github.com/nbittich/wtm/services/utils"
	"github.com/nbittich/wtm/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	PlanningCollection           = "planning"
	ProjectCollection            = "project"
	PlanningAssignmentCollection = "planningAssignment"
)

func GetPlanningAssignments(ctx context.Context, employeeID string, group types.Group) ([]types.PlanningAssignmentDetail, error) {
	collection, err := db.GetCollection(PlanningAssignmentCollection, group)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"employeeId": employeeID}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         PlanningCollection,
			"localField":   "entryId",
			"foreignField": "_id",
			"as":           "entry",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$entry",
			"preserveNullAndEmptyArrays": false,
		}}},
		{{Key: "$addFields", Value: bson.M{
			"entry": "$entry",
		}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         ProjectCollection,
			"localField":   "entry.projectId",
			"foreignField": "_id",
			"as":           "project",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$project",
			"preserveNullAndEmptyArrays": true,
		}}},
		{{Key: "$addFields", Value: bson.M{
			"project": "$project",
		}}},
	}
	return db.Aggregate[types.PlanningAssignmentDetail](ctx, collection, pipeline)
}

func GetProjects(ctx context.Context, group types.Group) ([]types.Project, error) {
	collection, err := db.GetCollection(ProjectCollection, group)
	if err != nil {
		return nil, err
	}
	return db.FindAll[types.Project](ctx, collection, nil)
}

func GetProject(ctx context.Context, projectID string, group types.Group) (*types.Project, error) {
	collection, err := db.GetCollection(ProjectCollection, group)
	if err != nil {
		return nil, err
	}

	project, err := db.FindOneByID[types.Project](ctx, collection, projectID)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func GetPlanning(ctx context.Context, projectID string, group types.Group) ([]types.PlanningEntry, error) {
	collection, err := db.GetCollection(PlanningCollection, group)
	if err != nil {
		return nil, err
	}

	filter := bson.M{
		"projectId": projectID,
	}
	return db.Find[types.PlanningEntry](ctx, &filter, collection, nil)
}

func AddOrUpdateProject(ctx context.Context, project *types.Project, group types.Group) (*types.Project, error) {
	if err := utils.ValidateStruct(project); err != nil {
		return nil, err
	}
	projectCollection, err := db.GetCollection(ProjectCollection, group)
	if err != nil {
		return nil, err
	}
	if project.ID != "" {
		project.UpdatedAt = time.Now()
	} else {
		project.CreatedAt = time.Now()
	}
	if _, err = db.InsertOrUpdate(ctx, project, projectCollection); err != nil {
		return nil, err
	}
	return project, nil
}

func AddOrUpdatePlanningEntry(ctx context.Context, entry *types.PlanningEntry, group types.Group) (*types.PlanningEntry, error) {
	if err := utils.ValidateStruct(entry); err != nil {
		return nil, err
	}
	var users []types.User
	var err error
	if len(entry.EmployeeIDs) != 0 {

		if len(entry.EmployeeIDs) > 1 && !entry.AllowMultipleAssignment {
			return nil, fmt.Errorf("multiple assignment is not allowed for this entry")
		}
		users, err = FindAllUsersByIDs(ctx, entry.EmployeeIDs, group)
		if err != nil {
			return nil, err
		}
		if len(users) != len(entry.EmployeeIDs) {
			return nil, fmt.Errorf("could not retrieve all employees")
		}
		for _, user := range users {
			if !user.Enabled || !slices.Contains(user.Roles, types.USER) {
				return nil, fmt.Errorf("user is not enabled or doesn't have the proper role")
			}
		}
	}

	projectCollection, err := db.GetCollection(ProjectCollection, group)
	if err != nil {
		return nil, err
	}
	project, err := db.FindOneByID[types.Project](ctx, projectCollection, entry.ProjectID)
	if err != nil {
		return nil, err
	}
	if project.Archived {
		return nil, fmt.Errorf("cannot create new planning entry on archived project")
	}
	now := time.Now()
	if entry.ID == "" {
		entry.CreatedAt = now
	} else {
		entry.UpdatedAt = &now
	}

	planningCollection, err := db.GetCollection(PlanningCollection, group)
	if err != nil {
		return nil, err
	}
	if _, err := db.InsertOrUpdate(ctx, entry, planningCollection); err != nil {
		return entry, err
	}
	go assignOrUnassignPlanningEntry(entry, users, project, group)

	return entry, nil
}

func MakePlanningCycle(ctx context.Context, cycle *types.PlanningCycle, group types.Group) ([]types.PlanningEntry, error) {
	if err := utils.ValidateStruct(cycle); err != nil {
		return nil, err
	}
	var users []types.User
	var err error
	if len(cycle.EmployeeIDs) != 0 {

		if len(cycle.EmployeeIDs) > 1 && !cycle.AllowMultipleAssignment {
			return nil, fmt.Errorf("multiple assignment is not allowed for this entry")
		}
		users, err = FindAllUsersByIDs(ctx, cycle.EmployeeIDs, group)
		if err != nil {
			return nil, err
		}
		if len(users) != len(cycle.EmployeeIDs) {
			return nil, fmt.Errorf("could not retrieve all employees")
		}
		for _, user := range users {
			if !user.Enabled || !slices.Contains(user.Roles, types.USER) {
				return nil, fmt.Errorf("user is not enabled or doesn't have the proper role")
			}
		}
	}
	startDay, err := time.Parse("02/01/2006", cycle.Start)
	if err != nil {
		return nil, err
	}
	endDay, err := time.Parse("02/01/2006", cycle.End)
	if err != nil {
		return nil, err
	}
	if startDay.After(endDay) {
		return nil, fmt.Errorf("start day cannot be after end day")
	}

	entry := &types.PlanningEntry{
		ProjectID:               cycle.ProjectID,
		EmployeeIDs:             cycle.EmployeeIDs,
		AllowMultipleAssignment: cycle.AllowMultipleAssignment,
		Title:                   cycle.Title,
		Description:             cycle.Description,
		Comments:                []types.Comment{},
	}

	dates := make([]time.Time, 0, 10)

	for d := startDay; !d.After(endDay); d.AddDate(0, 0, 1) {
		weekDay := d.Weekday()
		if (weekDay == time.Saturday && !cycle.IncludeSaturday) || (weekDay == time.Sunday && !cycle.IncludeSunday) {
			continue
		}
		dates = append(dates, d)
	}
	var frequency int
	switch cycle.RotationFrequencyType {
	case types.Days:
		frequency = int(cycle.RotationFrequency)
	case types.Weeks:
		frequency = int(cycle.RotationFrequency) * 7
	default:
		return nil, fmt.Errorf("unknown rotation frequency type")
	}
	shiftIndex := -1 // we want to start at 0
	var wg sync.WaitGroup

	ch := make(chan types.PlanningEntry, 2)
	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for idx, date := range dates {
		if idx%frequency == 0 {
			// new cycle
			shiftIndex += 1
			if shiftIndex == len(cycle.Shifts) {
				// reset shiftIndex
				shiftIndex = 0
			}
		}
		shift := cycle.Shifts[shiftIndex]
		start := time.Date(date.Year(), date.Month(), date.Day(), shift.StartHour, shift.StartMinute, 0, 0, date.Location())
		end := time.Date(date.Year(), date.Month(), date.Day(), shift.EndHour, shift.EndMinute, 0, 0, date.Location())
		entry.ID = ""
		entry.Start = start
		entry.End = end
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, ch chan<- types.PlanningEntry, errCh chan<- error, entry types.PlanningEntry, group types.Group) {
			defer wg.Done()
			entry.CreatedAt = time.Now()
			_, err := AddOrUpdatePlanningEntry(ctx, &entry, group)
			if err != nil {
				errCh <- err
			} else {
				ch <- entry
			}
		}(ctx, &wg, ch, errCh, *entry, group)

	}
	go func() {
		wg.Wait()
		close(ch)
		close(errCh)
	}()

	entries := make([]types.PlanningEntry, 0, len(dates))

	var errored error = nil
	for {
		select {
		case err, ok := <-errCh:
			if ok {
				errored = err
				cancel()
			}
		case entry, ok := <-ch:
			if ok {
				entries = append(entries, entry)
			}

		}
		if errCh == nil && ch == nil {
			break
		}
	}
	return entries, errored
}

func assignOrUnassignPlanningEntry(entry *types.PlanningEntry, users []types.User, project types.Project, group types.Group) {
	assignmentCol, err := db.GetCollection(PlanningAssignmentCollection, group)
	if len(users) == 0 {
		return
	}
	if err != nil {
		log.Println("could not get the planning collection", err)
		return
	}
	filter := bson.M{
		"entryId":   entry.ID,
		"cancelled": false,
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.MongoCtxTimeout)
	defer cancel()
	existingAssignements, err := db.Find[types.PlanningAssignment](ctx, filter, assignmentCol, nil)
	if err != nil {
		log.Println("could not fetch existing assignments", err)
		return
	}
	assignmentsToUpdate := make([]types.Identifiable, 0, len(existingAssignements))
	filteredUsersNewAssign := make([]*types.User, 0, len(users))
	filteredUsersCancelledAssign := make([]string, 0, len(users))
	for _, assignment := range existingAssignements {
		if !slices.Contains(entry.EmployeeIDs, assignment.EmployeeID) {
			assignment.Cancelled = true
			assignment.UpdatedAt = time.Now()
			assignmentsToUpdate = append(assignmentsToUpdate, &assignment)
			filteredUsersCancelledAssign = append(filteredUsersCancelledAssign, assignment.EmployeeID)
		}
	}
	usersToBeCancelled, err := FindAllUsersByIDs(ctx, filteredUsersCancelledAssign, group)
	if err != nil {
		log.Println("could not get users to unassign them", err)
		return
	}

	for _, user := range users {
		if !slices.ContainsFunc(existingAssignements, func(a types.PlanningAssignment) bool {
			return a.EmployeeID == user.ID
		}) {
			filteredUsersNewAssign = append(filteredUsersNewAssign, &user)
			assignmentsToUpdate = append(assignmentsToUpdate, &types.PlanningAssignment{
				EntryID:    entry.ID,
				EmployeeID: user.ID,
				CreatedAt:  time.Now(),
				SendDate:   time.Now(),
				Cancelled:  false,
			})
		}
	}
	if err = db.InsertOrUpdateMany(ctx, assignmentsToUpdate, assignmentCol); err != nil {
		log.Println("Could not update assignments", err)
		return
	}

	for _, user := range usersToBeCancelled {
		go email.SendAsync([]string{user.Email}, []string{}, "Cancelled planning assignment",
			fmt.Sprintf(`A planning assignment has been cancelled for project %s. You've been unassigned for slot %s -> %s`,
				project.Name, entry.Start.Format("02/01/2006 15:04"), entry.End.Format("02/01/2006 15:04")))
	}
	for _, user := range filteredUsersNewAssign {
		go email.SendAsync([]string{user.Email}, []string{}, "Planning assignment",
			fmt.Sprintf(`A planning assignment has been added for project %s. You've been assigned for slot %s -> %s`,
				project.Name, entry.Start.Format("02/01/2006 15:04"), entry.End.Format("02/01/2006 15:04")))
	}
}
