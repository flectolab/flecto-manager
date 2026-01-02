package user

import (
	stdContext "context"
	"fmt"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/service"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

type CreateChangePasswordDBFn func(ctx *appContext.Context) (*gorm.DB, error)

var NewChangePasswordDB CreateChangePasswordDBFn = func(ctx *appContext.Context) (*gorm.DB, error) {
	return database.CreateDB(ctx)
}

func GetChangePasswordCmd(ctx *appContext.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "change-password",
		Short: "change password for user",
		RunE:  GetChangePasswordRunFn(ctx),
	}
	cmd.Flags().StringP("username", "u", "", "username")
	cmd.Flags().StringP("password", "p", "", "password")
	return cmd
}

func GetChangePasswordRunFn(appCtx *appContext.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		ctx := stdContext.Background()
		db, errDb := NewChangePasswordDB(appCtx)
		if errDb != nil {
			return errDb
		}

		jwtService := jwt.NewServiceJWT(&appCtx.Config.Auth.JWT)
		repos := repository.NewRepositories(db)
		services := service.NewServices(appCtx, repos, jwtService)

		username, err := cmd.Flags().GetString("username")
		if err != nil {
			return err
		}

		password, err := cmd.Flags().GetString("password")
		if err != nil {
			return err
		}

		if username == "" || password == "" {
			return fmt.Errorf("username and password cannot be empty")
		}

		user, err := services.User.GetByUsername(ctx, username)
		if err != nil {
			return err
		}

		return services.User.UpdatePassword(ctx, user.ID, password)
	}
}
