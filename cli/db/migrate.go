package db

import (
	"errors"
	"fmt"
	"strings"

	"github.com/flectolab/flecto-manager/migrations"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/database"
	"github.com/spf13/cobra"
)

// Migrator defines the interface for database migrations
type Migrator interface {
	Up() error
	Down() error
	Steps(n int) error
	Version() (version uint, dirty bool, err error)
}

// MigratorFactory creates a Migrator instance
type MigratorFactory func(ctx *appContext.Context) (Migrator, error)

// NewMigrator is the factory function to create a Migrator (injectable for testing)
var NewMigrator MigratorFactory = createMigrator

func GetMigrateCmd(ctx *appContext.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration commands",
	}
	cmd.AddCommand(
		getMigrateApplyCmd(ctx),
		getMigrateDownCmd(ctx),
		getMigrateStatusCmd(ctx),
	)
	return cmd
}

func createMigrator(ctx *appContext.Context) (Migrator, error) {
	// Enable multiStatements for migrations (needed to execute multiple SQL statements)
	if dsn, ok := ctx.Config.DB.Config["dsn"].(string); ok {
		if !strings.Contains(dsn, "multiStatements") {
			if strings.Contains(dsn, "?") {
				ctx.Config.DB.Config["dsn"] = dsn + "&multiStatements=true"
			} else {
				ctx.Config.DB.Config["dsn"] = dsn + "?multiStatements=true"
			}
		}
	}

	// Get GORM database connection
	gormDB, err := database.CreateDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB
	db, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	// Create source driver from embedded FS
	sourceDriver, err := iofs.New(migrations.MigrationsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to create source driver: %w", err)
	}
	// Note: Don't close sourceDriver here - the migrate instance needs it

	// Create database driver
	dbDriver, err := mysql.WithInstance(db, &mysql.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrator
	m, err := migrate.NewWithInstance("iofs", sourceDriver, "mysql", dbDriver)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return m, nil
}

func getMigrateApplyCmd(ctx *appContext.Context) *cobra.Command {
	var steps int

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := NewMigrator(ctx)
			if err != nil {
				return err
			}

			if steps > 0 {
				err = m.Steps(steps)
			} else {
				err = m.Up()
			}

			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("No pending migrations")
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to apply migrations: %w", err)
			}

			fmt.Println("Migrations applied successfully")
			return nil
		},
	}

	cmd.Flags().IntVarP(&steps, "steps", "n", 0, "Number of migrations to apply (0 = all)")
	return cmd
}

func getMigrateDownCmd(ctx *appContext.Context) *cobra.Command {
	var steps int

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Rollback migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := NewMigrator(ctx)
			if err != nil {
				return err
			}

			if steps > 0 {
				err = m.Steps(-steps)
			} else {
				err = m.Down()
			}

			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("No migrations to rollback")
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to rollback migrations: %w", err)
			}

			fmt.Println("Migrations rolled back successfully")
			return nil
		},
	}

	cmd.Flags().IntVarP(&steps, "steps", "n", 0, "Number of migrations to rollback (0 = all)")
	return cmd
}

func getMigrateStatusCmd(ctx *appContext.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := NewMigrator(ctx)
			if err != nil {
				return err
			}
			version, dirty, err := m.Version()
			if errors.Is(err, migrate.ErrNilVersion) {
				fmt.Println("Migration Status")
				fmt.Println("================")
				fmt.Println("No migrations applied yet")
				return nil
			}
			if err != nil {
				return fmt.Errorf("failed to get version: %w", err)
			}

			fmt.Println("Migration Status")
			fmt.Println("================")
			fmt.Printf("Current version: %d\n", version)
			if dirty {
				fmt.Println("Status: DIRTY (migration failed, manual fix required)")
			} else {
				fmt.Println("Status: OK")
			}

			return nil
		},
	}
}
