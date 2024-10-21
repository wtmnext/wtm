package superadmin

import (
	"context"
	"fmt"

	"github.com/nbittich/wtm/services"
	"github.com/nbittich/wtm/services/db"
	"github.com/nbittich/wtm/services/utils"
	"github.com/nbittich/wtm/types"
	"go.mongodb.org/mongo-driver/bson"
)

var adminOrgCollection = db.GetAdminCollection("organization")

func ListOrgs(ctx context.Context) ([]types.Organization, error) {
	return db.FindAll[types.Organization](ctx, adminOrgCollection, nil)
}

func AddOrUpdateOrg(ctx context.Context, form *types.OrganizationForm) (*types.Organization, error) {
	if err := utils.ValidateStruct(form); err != nil {
		return nil, err
	}

	filter := bson.M{
		"$or": []bson.M{
			{"_id": form.ID},
			{"group": form.Group},
		},
	}
	org, err := db.FindOneBy[*types.Organization](ctx, filter, adminOrgCollection)
	if err != nil {
		if form.NewUser == nil {
			return nil, fmt.Errorf("user cannot be null")
		}
		// create a new org
		org = &types.Organization{
			Group:          types.Group(form.Group),
			FullName:       form.FullName,
			AdditionalInfo: form.AdditionalInfo,
			Email:          form.NewUser.Email,
		}
		if err := db.NewGroup(ctx, org.Group); err != nil {
			return org, err
		}
		if _, err := db.InsertOrUpdate(ctx, org, adminOrgCollection); err != nil {
			return org, err
		}
		// create a default user
		if form.NewUser.Role == nil {
			role := types.ADMIN
			form.NewUser.Role = &role
		}
		if _, err := services.NewUser(ctx, form.NewUser, org.Group); err != nil {
			return org, err
		}

		return org, nil
	}
	if org.ID == form.ID && org.Group == types.Group(form.Group) {
		org.FullName = form.FullName
		org.AdditionalInfo = form.AdditionalInfo
		if form.Email != nil {
			org.Email = *form.Email
		}
		if _, err := db.InsertOrUpdate(ctx, org, adminOrgCollection); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("group and id don't match")
	}
	return org, nil
}
