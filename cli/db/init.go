package db

import (
	stdContext "context"
	"fmt"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/service"
	"github.com/flectolab/flecto-manager/types"
	"github.com/flectolab/flecto-manager/validator"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func GetInitCmd(ctx *context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "init database",
		RunE:  GetInitRunFn(ctx),
	}
}

func GetInitRunFn(ctx *context.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		db, errDb := database.CreateDB(ctx)
		if errDb != nil {
			return errDb
		}

		err := initDatabase(db)
		if err != nil {
			return err
		}
		initData(db)
		return nil
	}
}

func initDatabase(db *gorm.DB) error {
	db.DisableForeignKeyConstraintWhenMigrating = true
	err := db.AutoMigrate(database.Models...)
	if err != nil {
		return fmt.Errorf("DB: failed to migrate database: %v", err)
	}
	db.DisableForeignKeyConstraintWhenMigrating = false
	err = db.AutoMigrate(database.Models...)
	if err != nil {
		return fmt.Errorf("DB: failed to migrate database: %v", err)
	}

	db.Exec(`ALTER TABLE projects ADD CONSTRAINT fk_project_namespace 
         FOREIGN KEY (namespace_code) REFERENCES namespaces(namespace_code) 
         ON DELETE CASCADE`)

	db.Exec(`ALTER TABLE redirects ADD CONSTRAINT fk_redirect_namespace_project 
         FOREIGN KEY (namespace_code, project_code) REFERENCES projects(namespace_code, project_code) 
         ON DELETE CASCADE`)

	db.Exec(`ALTER TABLE redirect_drafts ADD CONSTRAINT fk_redirect_draft_namespace_project
         FOREIGN KEY (namespace_code, project_code) REFERENCES projects(namespace_code, project_code)
         ON DELETE CASCADE`)

	db.Exec(`ALTER TABLE pages ADD CONSTRAINT fk_page_namespace_project
         FOREIGN KEY (namespace_code, project_code) REFERENCES projects(namespace_code, project_code)
         ON DELETE CASCADE`)

	db.Exec(`ALTER TABLE page_drafts ADD CONSTRAINT fk_page_draft_namespace_project
         FOREIGN KEY (namespace_code, project_code) REFERENCES projects(namespace_code, project_code)
         ON DELETE CASCADE`)

	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_redirects_source_unique ON redirects(namespace_code, project_code, source)`)
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_redirect_drafts_source_unique ON redirect_drafts(namespace_code, project_code, new_source)`)
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_pages_path_unique ON pages(namespace_code, project_code, path)`)
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_page_drafts_path_unique ON page_drafts(namespace_code, project_code, new_path)`)
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_agents_namespace_project_name ON agents(namespace_code, project_code, name)`)
	return nil
}

func initData(db *gorm.DB) {
	ctx := stdContext.Background()
	fmt.Println("Init database", time.Now())
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userService := service.NewUserService(validator.New(), userRepo, roleRepo)
	users := []*model.User{
		{Username: "admin", Lastname: "Admin", Firstname: "Admin", Active: types.Ptr(true)},
		{Username: "user", Lastname: "User", Firstname: "User", Active: types.Ptr(true)},
	}

	for i := 0; i < 30; i++ {
		username := fmt.Sprintf("user%d", i)
		user := &model.User{Username: username, Lastname: username, Firstname: username, Active: types.Ptr(i%2 == 0)}
		if i%2 == 0 {
			user.CreatedAt = time.Now().Add(-time.Hour*24*time.Duration(i) - time.Minute*time.Duration(i))
			user.UpdatedAt = time.Now().Add(-time.Hour * 24 * time.Duration(i))

		}
		users = append(users, user)
	}

	for i, user := range users {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Username), 12)
		user.Password = string(hashedPassword)
		users[i], _ = userService.Create(ctx, user)
	}

	namespaces := []model.Namespace{
		{NamespaceCode: "ns1", Name: "Namespace 1"},
		{NamespaceCode: "ns2", Name: "NS 2"},
		{NamespaceCode: "ns3", Name: "Namespace 3"},
	}
	for i := 0; i < 25; i++ {
		namespaces = append(namespaces, model.Namespace{NamespaceCode: fmt.Sprintf("ns_%d", i), Name: fmt.Sprintf("Namespace %d", i)})
	}
	errDb := gorm.G[model.Namespace](db).CreateInBatches(ctx, &namespaces, 100)
	if errDb != nil {
		panic(errDb)
	}

	projects := []model.Project{
		{ProjectCode: "prj1", Name: "Project 1", Namespace: &namespaces[0], NamespaceCode: namespaces[0].NamespaceCode},
		{ProjectCode: "prj2", Name: "Project 2", Namespace: &namespaces[0], NamespaceCode: namespaces[0].NamespaceCode},
		{ProjectCode: "prj3", Name: "Project 3", Namespace: &namespaces[0], NamespaceCode: namespaces[0].NamespaceCode},
		{ProjectCode: "prj1", Name: "Project 1", Namespace: &namespaces[1], NamespaceCode: namespaces[1].NamespaceCode},
		{ProjectCode: "prj2", Name: "Project 2", Namespace: &namespaces[1], NamespaceCode: namespaces[1].NamespaceCode},
		{ProjectCode: "prj3", Name: "Project 3", Namespace: &namespaces[1], NamespaceCode: namespaces[1].NamespaceCode},
	}

	for i := 0; i < 25; i++ {
		projects = append(projects, model.Project{NamespaceCode: fmt.Sprintf("ns_%d", i), ProjectCode: fmt.Sprintf("prj%d", i), Name: fmt.Sprintf("Project %d", i)})
	}
	errDb = gorm.G[model.Project](db).CreateInBatches(ctx, &projects, 100)
	if errDb != nil {
		panic(errDb)
	}

	redirects := []model.Redirect{}
	for i := 1; i < 40; i++ {
		redirects = append(redirects, model.Redirect{
			NamespaceCode: "ns1",
			ProjectCode:   "prj1",
			IsPublished:   types.Ptr(true),
			Redirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/project/" + fmt.Sprintf("%d", i),
				Target: "/catalog/product/" + fmt.Sprintf("%d", i),
				Status: commonTypes.RedirectStatusPermanent,
			},
		})
	}
	errDb = gorm.G[model.Redirect](db).CreateInBatches(ctx, &redirects, 100)
	if errDb != nil {
		panic(errDb)
	}

	content := fmt.Sprintf("Page robots.txt content")
	page := model.Page{
		NamespaceCode: "ns1",
		ProjectCode:   "prj1",
		IsPublished:   types.Ptr(true),
		ContentSize:   int64(len(content)),
		Page: &commonTypes.Page{
			Type:        commonTypes.PageTypeBasic,
			ContentType: commonTypes.PageContentTypeTextPlain,
			Path:        "/robots.txt",
			Content:     content,
		},
	}
	errDb = gorm.G[model.Page](db).Create(ctx, &page)
	if errDb != nil {
		panic(errDb)
	}

	fmt.Println("Finish database", time.Now())

	fmt.Println("Init Permissions", time.Now())
	roles := []model.Role{
		{Code: "admin", Type: model.RoleTypeRole},
		{Code: "editor", Type: model.RoleTypeRole},
		{Code: "ns2", Type: model.RoleTypeRole},
		{Code: "admin", Type: model.RoleTypeUser},
		{Code: "user", Type: model.RoleTypeUser},
	}
	_ = gorm.G[model.Role](db).CreateInBatches(ctx, &roles, 100)

	adminPerms := []model.AdminPermission{
		{Section: model.AdminSectionAll, Action: model.ActionAll, Role: roles[0]},
	}
	_ = gorm.G[model.AdminPermission](db).CreateInBatches(ctx, &adminPerms, 100)

	resourcesPerms := []model.ResourcePermission{
		{Namespace: "*", Project: "*", Action: model.ActionAll, Role: roles[0]},
		{Namespace: "*", Project: "*", Action: model.ActionAll, Role: roles[1]},
		{Namespace: "ns2", Project: "*", Action: model.ActionAll, Role: roles[2]},
		{Namespace: "ns1", Project: "*", Action: model.ActionRead, Role: roles[4]},
	}
	_ = gorm.G[model.ResourcePermission](db).CreateInBatches(ctx, &resourcesPerms, 100)

	_ = gorm.G[model.UserRole](db).Create(ctx, &model.UserRole{Role: roles[0], User: *users[0]})
	_ = gorm.G[model.UserRole](db).Create(ctx, &model.UserRole{Role: roles[3], User: *users[0]})
	_ = gorm.G[model.UserRole](db).Create(ctx, &model.UserRole{Role: roles[2], User: *users[1]})
	_ = gorm.G[model.UserRole](db).Create(ctx, &model.UserRole{Role: roles[4], User: *users[1]})

	_ = gorm.G[model.Agent](db).Create(ctx, &model.Agent{NamespaceCode: projects[0].NamespaceCode, ProjectCode: projects[0].ProjectCode, Agent: commonTypes.Agent{Name: "agent-1", Version: projects[0].Version, Type: commonTypes.AgentTypeTraefik, Status: commonTypes.AgentStatusSuccess, LoadDuration: 100 * time.Millisecond}})
	_ = gorm.G[model.Agent](db).Create(ctx, &model.Agent{NamespaceCode: projects[0].NamespaceCode, ProjectCode: projects[0].ProjectCode, Agent: commonTypes.Agent{Name: "agent-2", Version: projects[0].Version, Type: commonTypes.AgentTypeDefault, Status: commonTypes.AgentStatusError, Error: "error call api", LoadDuration: 100 * time.Millisecond}})

	fmt.Println("Finish Permissions", time.Now())
}
