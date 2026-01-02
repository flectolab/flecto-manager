package db

import (
	stdContext "context"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/flectolab/flecto-manager/hash"
	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// CreateInitDBFn is a function type for creating database connection (used for testing)
type CreateInitDBFn func(ctx *appContext.Context) (*gorm.DB, error)

// NewInitDB is the function used to create database connection (can be replaced in tests)
var NewInitDB CreateInitDBFn = func(ctx *appContext.Context) (*gorm.DB, error) {
	return database.CreateDB(ctx)
}

func GetInitCmd(ctx *appContext.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "init database",
		RunE:  GetInitRunFn(ctx),
	}
}

func GetInitRunFn(ctx *appContext.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		db, errDb := NewInitDB(ctx)
		if errDb != nil {
			return errDb
		}
		return initData(ctx, db)
	}
}

func initData(appCtx *appContext.Context, db *gorm.DB) error {
	ctx := stdContext.Background()

	jwtService := jwt.NewServiceJWT(&appCtx.Config.Auth.JWT)
	repos := repository.NewRepositories(db)
	services := service.NewServices(appCtx, repos, jwtService)

	adminUser := &model.User{Username: "admin", Lastname: "Admin", Firstname: "Admin", Active: types.Ptr(true)}
	hashedPassword, _ := hash.Password(adminUser.Username)
	adminUser.Password = string(hashedPassword)
	adminUser, err := services.User.Create(ctx, adminUser)
	if err != nil {
		return err
	}

	adminRole := &model.Role{
		Code: "admin",
		Type: model.RoleTypeRole,
		Resources: []model.ResourcePermission{
			{Namespace: "*", Project: "*", Action: model.ActionAll, Resource: model.ResourceTypeAll},
		},
		Admin: []model.AdminPermission{
			{Section: model.AdminSectionAll, Action: model.ActionAll},
		},
	}
	adminRole, err = services.Role.Create(ctx, adminRole)
	if err != nil {
		return err
	}
	err = services.Role.AddUserToRole(ctx, adminUser.ID, adminRole.ID)
	if err != nil {
		return err
	}
	return nil
}
