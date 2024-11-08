package project

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nbittich/wtm/config"
	"github.com/nbittich/wtm/services"
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

func IsUserAvailable(ctx context.Context, employeeID string, entry *types.PlanningEntry, group types.Group) (bool, error) {
	var (
		user                                   types.User
		err                                    error
		assignedStart, assignedEnd, start, end time.Time
		ok                                     bool
	)
	if user, err = services.FindUserByID(ctx, employeeID, group); err != nil {
		return false, err
	}
	if user.Profile.Availability != nil {
		if ok, err = user.Profile.Availability.IsAvailable(entry.Start, entry.End); err != nil || !ok {
			return ok, err
		}
	}
	details, err := GetPlanningAssignments(ctx, employeeID, group)
	if err != nil {
		return false, err
	}
	for _, detail := range details {
		if detail.Entry.ID == entry.ID {
			continue
		}
		if assignedStart, err = time.Parse(types.BelgianDateTimeFormat, detail.Entry.Start); err != nil {
			return false, err
		}
		if assignedEnd, err = time.Parse(types.BelgianDateTimeFormat, detail.Entry.End); err != nil {
			return false, err
		}
		if start, err = time.Parse(types.BelgianDateTimeFormat, entry.Start); err != nil {
			return false, err
		}
		if end, err = time.Parse(types.BelgianDateTimeFormat, entry.End); err != nil {
			return false, err
		}
		if !(end.Before(assignedStart) || start.After(assignedEnd)) {
			return false, nil
		}
	}

	return true, nil
}

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

func AddOrUpdatePlanningEntry(ctx context.Context, entry types.PlanningEntry, assign bool, group types.Group) (*types.PlanningEntry, error) {
	if err := utils.ValidateStruct(entry); err != nil {
		return nil, err
	}
	var users []types.User
	var err error
	if len(entry.EmployeeIDs) != 0 {

		if len(entry.EmployeeIDs) > 1 && !entry.AllowMultipleAssignment {
			return nil, fmt.Errorf("multiple assignment is not allowed for this entry")
		}
		users, err = services.FindAllUsersByIDs(ctx, entry.EmployeeIDs, group)
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
	if _, err := db.InsertOrUpdate(ctx, &entry, planningCollection); err != nil {
		return &entry, err
	}
	if assign {
		go func() {
			if result, err := assignOrUnassignPlanningEntry(entry, project, group); err == nil {
				sendMailAssignOrUnassign([]planningAssignmentResult{*result})
			} else {
				log.Println("could not assign/unassign planning entry")
			}
		}()
	}

	return &entry, nil
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
		users, err = services.FindAllUsersByIDs(ctx, cycle.EmployeeIDs, group)
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
	startDay, err := time.Parse(types.BelgianDateFormat, cycle.Start)
	if err != nil {
		return nil, err
	}
	endDay, err := time.Parse(types.BelgianDateFormat, cycle.End)
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

	project, err := GetProject(ctx, cycle.ProjectID, group)
	if err != nil {
		return nil, err
	}
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
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

		var extraDay int
		// check if is between two dates
		if shift.EndHour < shift.StartHour {
			extraDay += 1
		}
		start := time.Date(date.Year(), date.Month(), date.Day(), shift.StartHour, shift.StartMinute, 0, 0, date.Location())
		end := time.Date(date.Year(), date.Month(), date.Day()+extraDay, shift.EndHour, shift.EndMinute, 0, 0, date.Location())
		entry.ID = ""

		entry.Start = start.Format(types.BelgianDateTimeFormat)
		entry.End = end.Format(types.BelgianDateTimeFormat)
		wg.Add(1)
		go func(ctx context.Context, wg *sync.WaitGroup, ch chan<- types.PlanningEntry, errCh chan<- error, entry types.PlanningEntry, group types.Group) {
			defer wg.Done()
			entry.CreatedAt = time.Now()
			newEntry, err := AddOrUpdatePlanningEntry(ctx, entry, false, group)
			if err != nil {
				errCh <- err
			} else {
				ch <- *newEntry
			}
		}(ctx, &wg, ch, errCh, *entry, group)

	}

	go func() {
		wg.Wait()
		close(ch)
		close(errCh)
		ch = nil
		errCh = nil
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

	// assigned and send mail
	go func(entries []types.PlanningEntry, project types.Project, group types.Group) {
		assignmentResults := make([]planningAssignmentResult, 0, len(entries))
		for _, entry := range entries {
			result, err := assignOrUnassignPlanningEntry(entry, project, group)
			if err != nil {
				log.Println("could not assign... Err:", err, "no mail sent! entries:", entries)
				return
			}
			assignmentResults = append(assignmentResults, *result)
		}
		sendMailAssignOrUnassign(assignmentResults)
	}(entries, *project, group)

	return entries, errored
}

type planningAssignmentResult struct {
	usersToBeCancelled     []types.User
	filteredUsersNewAssign []types.User
	entry                  types.PlanningEntry
	project                types.Project
}

func sendMailAssignOrUnassign(assignmentResults []planningAssignmentResult) {
	type UserKey struct {
		UserID string
		Email  string
	}
	usersToBecancelled := make(map[UserKey][]string)
	usersNewAssign := make(map[UserKey][]string)
	slices.SortFunc(assignmentResults, func(a planningAssignmentResult, b planningAssignmentResult) int {
		start, err := time.Parse(types.BelgianDateTimeFormat, a.entry.Start)
		if err != nil {
			log.Println("could not parse start date...sorting will be wrong", err)
			return 0
		}
		end, err := time.Parse(types.BelgianDateTimeFormat, b.entry.Start)
		if err != nil {
			log.Println("could not parse end date...sorting will be wrong", err)
			return 0
		}
		return start.Compare(end)
	})
	for _, assignmentResult := range assignmentResults {
		for _, user := range assignmentResult.usersToBeCancelled {
			userKey := UserKey{UserID: user.ID, Email: user.Email}
			if _, exists := usersToBecancelled[userKey]; !exists {
				usersToBecancelled[userKey] = make([]string, 0, 10)
			}
			usersToBecancelled[userKey] = append(usersToBecancelled[userKey],
				fmt.Sprintf(`Project %s: You've been unassigned for slot %s -> %s`,
					assignmentResult.project.Name, assignmentResult.entry.Start, assignmentResult.entry.End))

		}
		for _, user := range assignmentResult.filteredUsersNewAssign {
			userKey := UserKey{UserID: user.ID, Email: user.Email}
			if _, exists := usersNewAssign[userKey]; !exists {
				usersNewAssign[userKey] = make([]string, 0, 10)
			}
			usersNewAssign[userKey] = append(usersNewAssign[userKey],
				fmt.Sprintf(`Project %s: You've been assigned for slot %s -> %s`,
					assignmentResult.project.Name, assignmentResult.entry.Start, assignmentResult.entry.End))

		}
	}

	for user, messages := range usersToBecancelled {
		email.SendAsync([]string{user.Email}, []string{}, "[CANCELLED]: Planning assignment(s)",
			strings.Join(messages, "<br>"))
	}
	for user, messages := range usersNewAssign {
		email.SendAsync([]string{user.Email}, []string{}, "Planning assignment(s)",
			strings.Join(messages, "<br>"))
	}
}

func assignOrUnassignPlanningEntry(entry types.PlanningEntry, project types.Project, group types.Group) (*planningAssignmentResult, error) {
	assignmentCol, err := db.GetCollection(PlanningAssignmentCollection, group)
	if err != nil {
		log.Println("could not get the assignment collection", err)
		return nil, err
	}
	planningCol, err := db.GetCollection(PlanningCollection, group)
	if err != nil {
		log.Println("could not get the planning collection", err)
		return nil, err
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
		return nil, err
	}
	// delete employee ids that are not available
	removedEmployeeIds := make([]string, 0, len(entry.EmployeeIDs))
	entry.EmployeeIDs = slices.DeleteFunc(entry.EmployeeIDs, func(id string) bool {
		ok, err := IsUserAvailable(ctx, id, &entry, group)
		if err != nil {
			log.Println("could not check if user available", err, entry)
		}
		if !ok {
			removedEmployeeIds = append(removedEmployeeIds, id)
		}
		return !ok
	})
	if len(removedEmployeeIds) != 0 {
		users, err := services.FindAllUsersByIDs(ctx, removedEmployeeIds, group)
		if err != nil {
			log.Println("could not get all users", err)
		}
		for _, user := range users {
			entry.Comments = append(entry.Comments, types.Comment{
				UserID:      user.ID,
				Message:     fmt.Sprintf("Could not assign %s. Employee is already assigned to another project or doesn't work at that time", user.Username),
				CommentType: types.WARNING,
				CreatedAt:   time.Now(),
				UpdatedAt:   nil,
			})
		}
		db.InsertOrUpdate(ctx, &entry, planningCol)
	}
	assignmentsToUpdate := make([]types.Identifiable, 0, len(existingAssignements))
	filteredUsersNewAssign := make([]*types.User, 0, len(existingAssignements))
	filteredUsersCancelledAssign := make([]string, 0, len(existingAssignements))
	for _, assignment := range existingAssignements {
		if !slices.Contains(entry.EmployeeIDs, assignment.EmployeeID) {
			assignment.Cancelled = true
			assignment.UpdatedAt = time.Now()
			assignmentsToUpdate = append(assignmentsToUpdate, &assignment)
			filteredUsersCancelledAssign = append(filteredUsersCancelledAssign, assignment.EmployeeID)
		}
	}
	usersToBeCancelled, err := services.FindAllUsersByIDs(ctx, filteredUsersCancelledAssign, group)
	if err != nil {
		log.Println("could not get users to unassign them", err)
		return nil, err
	}

	users, err := services.FindAllUsersByIDs(ctx, entry.EmployeeIDs, group)
	if err != nil {
		log.Println("could not get users to assign them", err)
		return nil, err
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
		return nil, err
	}
	return &planningAssignmentResult{
		usersToBeCancelled:     usersToBeCancelled,
		filteredUsersNewAssign: users,
		entry:                  entry,
		project:                project,
	}, nil
}
