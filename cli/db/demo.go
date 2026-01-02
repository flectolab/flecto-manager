package db

import (
	stdContext "context"
	"fmt"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/flectolab/flecto-manager/jwt"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// CreateDemoDBFn is a function type for creating database connection (used for testing)
type CreateDemoDBFn func(ctx *appContext.Context) (*gorm.DB, error)

// NewDemoDB is the function used to create database connection (can be replaced in tests)
var NewDemoDB CreateDemoDBFn = func(ctx *appContext.Context) (*gorm.DB, error) {
	return database.CreateDB(ctx)
}

func GetDemoCmd(ctx *appContext.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "demo",
		Short: "add demo data",
		RunE:  GetDemoRunFn(ctx),
	}
}

func GetDemoRunFn(ctx *appContext.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		db, errDb := NewDemoDB(ctx)
		if errDb != nil {
			return errDb
		}
		return demoData(ctx, db)
	}
}

func demoData(appCtx *appContext.Context, db *gorm.DB) error {
	ctx := stdContext.Background()

	jwtService := jwt.NewServiceJWT(&appCtx.Config.Auth.JWT)
	repos := repository.NewRepositories(db)
	services := service.NewServices(appCtx, repos, jwtService)

	namespace1 := &model.Namespace{NamespaceCode: "ns1", Name: "Namespace 1"}
	namespace2 := &model.Namespace{NamespaceCode: "ns2", Name: "Namespace 2"}

	namespaces := []*model.Namespace{
		namespace1,
		namespace2,
	}
	for i, namespace := range namespaces {
		ns, err := services.Namespace.Create(ctx, namespace)
		if err != nil {
			return err
		}
		namespaces[i] = ns
	}

	projects := []*model.Project{
		{ProjectCode: "prj1", Name: "Project 1", Namespace: namespace1, NamespaceCode: namespace1.NamespaceCode},
		{ProjectCode: "prj2", Name: "Project 2", Namespace: namespace1, NamespaceCode: namespace1.NamespaceCode},
		{ProjectCode: "prj3", Name: "Project 3", Namespace: namespace1, NamespaceCode: namespace1.NamespaceCode},
		{ProjectCode: "prj1", Name: "Project 1", Namespace: namespace2, NamespaceCode: namespace2.NamespaceCode},
		{ProjectCode: "prj2", Name: "Project 2", Namespace: namespace2, NamespaceCode: namespace2.NamespaceCode},
		{ProjectCode: "prj3", Name: "Project 3", Namespace: namespace2, NamespaceCode: namespace2.NamespaceCode},
	}

	for i, project := range projects {
		prj, err := services.Project.Create(ctx, project)
		if err != nil {
			return err
		}
		projects[i] = prj
	}

	redirects := []*model.Redirect{}
	for i := 1; i < 40; i++ {
		redirects = append(redirects, &model.Redirect{
			NamespaceCode: projects[0].NamespaceCode,
			ProjectCode:   projects[0].ProjectCode,
			IsPublished:   types.Ptr(true),
			PublishedAt:   time.Now(),
			Redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/project/" + fmt.Sprintf("%d", i),
				Target: "/catalog/product/" + fmt.Sprintf("%d", i),
				Status: commonTypes.RedirectStatusPermanent,
			},
		})
	}

	for _, redirect := range redirects {
		_, err := services.RedirectDraft.Create(ctx, redirect.NamespaceCode, redirect.ProjectCode, nil, redirect.Redirect)
		if err != nil {
			return err
		}
	}

	content := fmt.Sprintf("Page robots.txt content")
	page := model.Page{
		NamespaceCode: projects[0].NamespaceCode,
		ProjectCode:   projects[0].ProjectCode,
		IsPublished:   types.Ptr(true),
		PublishedAt:   time.Now(),
		ContentSize:   int64(len(content)),
		Page: &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			ContentType: commonTypes.PageContentTypeTextPlain,
			Path:        "/robots.txt",
			Content:     content,
		},
	}
	_, err := services.PageDraft.Create(ctx, page.NamespaceCode, page.ProjectCode, nil, page.Page)
	if err != nil {
		return err
	}

	_, err = services.Project.Publish(ctx, projects[0].NamespaceCode, projects[0].ProjectCode)
	if err != nil {
		return err
	}

	return nil
}
