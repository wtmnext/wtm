package services

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/nbittich/wtm/config"
	"github.com/nbittich/wtm/services/db"
	"github.com/nbittich/wtm/services/email"
	"github.com/nbittich/wtm/services/utils"
	"github.com/nbittich/wtm/types"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	PlanningCollection           = "planning"
	ProjectCollection            = "project"
	PlanningAssignmentCollection = "planningAssignment"
)

func GetProjects(ctx context.Context, group types.Group) ([]types.Project, error) {
	collection, err := db.GetCollection(ProjectCollection, group)
	if err != nil {
		return nil, err
	}
	return db.FindAll[types.Project](ctx, collection, nil)
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
	if entry.ID == "" {
		entry.CreatedAt = time.Now()
	} else {
		entry.UpdatedAt = time.Now()
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

func assignOrUnassignPlanningEntry(entry *types.PlanningEntry, users []types.User, project types.Project, group types.Group) {
	collection, err := db.GetCollection(PlanningCollection, group)
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
	existingAssignements, err := db.Find[types.PlanningAssignment](ctx, filter, collection, nil)
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
	if err = db.InsertOrUpdateMany(ctx, assignmentsToUpdate, collection); err != nil {
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
