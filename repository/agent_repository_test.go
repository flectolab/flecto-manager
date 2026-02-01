package repository

import (
	"context"
	"testing"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAgentTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Agent{})
	assert.NoError(t, err)

	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_agents_namespace_project_name ON agents(namespace_code, project_code, name)`)

	return db
}

func createTestAgentNamespace(t *testing.T, db *gorm.DB, code, name string) *model.Namespace {
	ns := &model.Namespace{
		NamespaceCode: code,
		Name:          name,
	}
	err := db.Create(ns).Error
	assert.NoError(t, err)
	return ns
}

func createTestAgentProject(t *testing.T, db *gorm.DB, namespaceCode, projectCode, name string) *model.Project {
	proj := &model.Project{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		Name:          name,
	}
	err := db.Create(proj).Error
	assert.NoError(t, err)
	return proj
}

func TestNewAgentRepository(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentRepository(db)

	assert.NotNil(t, repo)
}

func TestAgentRepository_GetTx(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentRepository(db)
	ctx := context.Background()

	tx := repo.GetTx(ctx)
	assert.NotNil(t, tx)

	// GetTx returns a db session that can be used for transactions
	var agents []model.Agent
	err := tx.Find(&agents).Error
	assert.NoError(t, err)
}

func TestAgentRepository_GetQuery(t *testing.T) {
	db := setupAgentTestDB(t)
	repo := NewAgentRepository(db)
	ctx := context.Background()

	query := repo.GetQuery(ctx)
	assert.NotNil(t, query)

	var agents []model.Agent
	err := query.Find(&agents).Error
	assert.NoError(t, err)
}

func TestAgentRepository_Upsert(t *testing.T) {
	t.Run("insert new agent", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-1",
				Type:         commonTypes.AgentTypeTraefik,
				Status:       commonTypes.AgentStatusSuccess,
				LoadDuration: commonTypes.NewDuration(100 * time.Millisecond),
			},
		}

		err := repo.Upsert(ctx, agent)

		assert.NoError(t, err)
		assert.NotZero(t, agent.ID)

		var found model.Agent
		db.First(&found, agent.ID)
		assert.Equal(t, "agent-1", found.Name)
		assert.Equal(t, commonTypes.AgentTypeTraefik, found.Type)
		assert.Equal(t, commonTypes.AgentStatusSuccess, found.Status)
	})

	t.Run("insert missing status returns error", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-1",
				Type:         commonTypes.AgentTypeTraefik,
				LoadDuration: commonTypes.NewDuration(100 * time.Millisecond),
			},
		}

		err := repo.Upsert(ctx, agent)

		assert.ErrorIs(t, err, ErrAgentMissingStatus)
	})

	t.Run("insert missing type returns error", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-1",
				Status:       commonTypes.AgentStatusSuccess,
				LoadDuration: commonTypes.NewDuration(100 * time.Millisecond),
			},
		}

		err := repo.Upsert(ctx, agent)

		assert.ErrorIs(t, err, ErrAgentMissingType)
	})

	t.Run("insert missing load_duration returns error", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		}

		err := repo.Upsert(ctx, agent)

		assert.ErrorIs(t, err, ErrAgentMissingLoadDuration)
	})

	t.Run("update existing agent replaces all fields", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-1",
				Type:         commonTypes.AgentTypeTraefik,
				Status:       commonTypes.AgentStatusSuccess,
				LoadDuration: commonTypes.NewDuration(100 * time.Millisecond),
				Version:      1,
			},
		}
		err := repo.Upsert(ctx, agent)
		assert.NoError(t, err)
		originalID := agent.ID

		updatedAgent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-1",
				Type:         commonTypes.AgentTypeDefault,
				Status:       commonTypes.AgentStatusError,
				LoadDuration: commonTypes.NewDuration(200 * time.Millisecond),
				Error:        "connection timeout",
				Version:      2,
			},
		}
		err = repo.Upsert(ctx, updatedAgent)

		assert.NoError(t, err)

		var found model.Agent
		db.First(&found, originalID)
		assert.Equal(t, commonTypes.AgentStatusError, found.Status)
		assert.Equal(t, commonTypes.AgentTypeDefault, found.Type)
		assert.Equal(t, "connection timeout", found.Error)
		assert.Equal(t, commonTypes.NewDuration(200*time.Millisecond), found.LoadDuration)
		assert.Equal(t, 2, found.Version)

		var count int64
		db.Model(&model.Agent{}).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("update sets agent ID", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-1",
				Type:         commonTypes.AgentTypeTraefik,
				Status:       commonTypes.AgentStatusSuccess,
				LoadDuration: commonTypes.NewDuration(100 * time.Millisecond),
			},
		}
		err := repo.Upsert(ctx, agent)
		assert.NoError(t, err)
		originalID := agent.ID

		updatedAgent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-1",
				Type:         commonTypes.AgentTypeTraefik,
				Status:       commonTypes.AgentStatusError,
				LoadDuration: commonTypes.NewDuration(200 * time.Millisecond),
			},
		}
		assert.Zero(t, updatedAgent.ID)

		err = repo.Upsert(ctx, updatedAgent)

		assert.NoError(t, err)
		assert.Equal(t, originalID, updatedAgent.ID)
	})

	t.Run("insert different agents for same project", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent1 := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-1",
				Type:         commonTypes.AgentTypeTraefik,
				Status:       commonTypes.AgentStatusSuccess,
				LoadDuration: commonTypes.NewDuration(100 * time.Millisecond),
			},
		}
		agent2 := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-2",
				Type:         commonTypes.AgentTypeTraefik,
				Status:       commonTypes.AgentStatusSuccess,
				LoadDuration: commonTypes.NewDuration(100 * time.Millisecond),
			},
		}

		err := repo.Upsert(ctx, agent1)
		assert.NoError(t, err)
		err = repo.Upsert(ctx, agent2)
		assert.NoError(t, err)

		var count int64
		db.Model(&model.Agent{}).Count(&count)
		assert.Equal(t, int64(2), count)
	})
}

func TestAgentRepository_FindByName(t *testing.T) {
	t.Run("find existing agent", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		}
		db.Create(agent)

		found, err := repo.FindByName(ctx, "test-ns", "test-proj", "agent-1")

		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "agent-1", found.Name)
	})

	t.Run("agent not found", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		found, err := repo.FindByName(ctx, "test-ns", "test-proj", "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, found)
	})

	t.Run("agent wrong namespace", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentNamespace(t, db, "other-ns", "Other Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		}
		db.Create(agent)

		found, err := repo.FindByName(ctx, "other-ns", "test-proj", "agent-1")

		assert.Error(t, err)
		assert.Nil(t, found)
	})
}

func TestAgentRepository_FindByProject(t *testing.T) {
	t.Run("returns agents for project", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			db.Create(&model.Agent{
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				Agent: commonTypes.Agent{
					Name:   "agent-" + string(rune('a'+i)),
					Type:   commonTypes.AgentTypeTraefik,
					Status: commonTypes.AgentStatusSuccess,
				},
			})
		}

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("returns empty slice when no agents", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		results, err := repo.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("only returns agents for specified project", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "proj-a", "Project A")
		createTestAgentProject(t, db, "test-ns", "proj-b", "Project B")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "proj-a",
			Agent: commonTypes.Agent{
				Name:   "agent-a",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		})
		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "proj-b",
			Agent: commonTypes.Agent{
				Name:   "agent-b",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		})

		results, err := repo.FindByProject(ctx, "test-ns", "proj-a")

		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "agent-a", results[0].Name)
	})
}

func TestAgentRepository_SearchPaginate(t *testing.T) {
	db := setupAgentTestDB(t)
	createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
	createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
	repo := NewAgentRepository(db)
	ctx := context.Background()

	for i := 0; i < 15; i++ {
		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-" + string(rune('a'+i)),
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		})
	}

	tests := []struct {
		name      string
		query     *gorm.DB
		limit     int
		offset    int
		wantCount int
		wantTotal int64
	}{
		{
			name:      "paginate with limit",
			query:     nil,
			limit:     5,
			offset:    0,
			wantCount: 5,
			wantTotal: 15,
		},
		{
			name:      "paginate with offset",
			query:     nil,
			limit:     5,
			offset:    10,
			wantCount: 5,
			wantTotal: 15,
		},
		{
			name:      "paginate with offset beyond total",
			query:     nil,
			limit:     5,
			offset:    20,
			wantCount: 0,
			wantTotal: 15,
		},
		{
			name:      "paginate without limit returns all",
			query:     nil,
			limit:     0,
			offset:    0,
			wantCount: 15,
			wantTotal: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, total, err := repo.SearchPaginate(ctx, tt.query, tt.limit, tt.offset)

			assert.NoError(t, err)
			assert.Len(t, results, tt.wantCount)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}

func TestAgentRepository_CountByProjectAndStatus(t *testing.T) {
	t.Run("count agents with error status", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		now := time.Now()
		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			LastHitAt:     now,
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		})
		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			LastHitAt:     now,
			Agent: commonTypes.Agent{
				Name:   "agent-2",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusError,
			},
		})
		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			LastHitAt:     now,
			Agent: commonTypes.Agent{
				Name:   "agent-3",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusError,
			},
		})

		count, err := repo.CountByProjectAndStatus(ctx, "test-ns", "test-proj", commonTypes.AgentStatusError, time.Now().Add(-time.Hour))

		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})

	t.Run("count agents with success status", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		now := time.Now()
		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			LastHitAt:     now,
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		})
		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			LastHitAt:     now,
			Agent: commonTypes.Agent{
				Name:   "agent-2",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusError,
			},
		})

		count, err := repo.CountByProjectAndStatus(ctx, "test-ns", "test-proj", commonTypes.AgentStatusSuccess, time.Now().Add(-time.Hour))

		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("returns zero when no agents match", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			LastHitAt:     time.Now(),
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		})

		count, err := repo.CountByProjectAndStatus(ctx, "test-ns", "test-proj", commonTypes.AgentStatusError, time.Now().Add(-time.Hour))

		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})

	t.Run("filters by namespace and project", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "ns-a", "Namespace A")
		createTestAgentNamespace(t, db, "ns-b", "Namespace B")
		createTestAgentProject(t, db, "ns-a", "proj-a", "Project A")
		createTestAgentProject(t, db, "ns-b", "proj-b", "Project B")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		now := time.Now()
		db.Create(&model.Agent{
			NamespaceCode: "ns-a",
			ProjectCode:   "proj-a",
			LastHitAt:     now,
			Agent: commonTypes.Agent{
				Name:   "agent-a",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusError,
			},
		})
		db.Create(&model.Agent{
			NamespaceCode: "ns-b",
			ProjectCode:   "proj-b",
			LastHitAt:     now,
			Agent: commonTypes.Agent{
				Name:   "agent-b",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusError,
			},
		})

		count, err := repo.CountByProjectAndStatus(ctx, "ns-a", "proj-a", commonTypes.AgentStatusError, time.Now().Add(-time.Hour))

		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("only counts agents with last_hit_at after threshold", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		oldTime := time.Now().Add(-2 * time.Hour)
		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			LastHitAt:     oldTime,
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusError,
			},
		})

		threshold := time.Now().Add(-time.Hour)

		newTime := time.Now()
		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			LastHitAt:     newTime,
			Agent: commonTypes.Agent{
				Name:   "agent-2",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusError,
			},
		})

		// Only agent-2 has last_hit_at after threshold
		count, err := repo.CountByProjectAndStatus(ctx, "test-ns", "test-proj", commonTypes.AgentStatusError, threshold)

		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)

		// Both agents have last_hit_at after 3 hours ago
		count, err = repo.CountByProjectAndStatus(ctx, "test-ns", "test-proj", commonTypes.AgentStatusError, time.Now().Add(-3*time.Hour))

		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
	})
}

func TestAgentRepository_UpdateLastHit(t *testing.T) {
	t.Run("updates last_hit_at for existing agent", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		}
		db.Create(agent)

		var originalAgent model.Agent
		db.First(&originalAgent, agent.ID)
		originalLastHit := originalAgent.LastHitAt

		time.Sleep(10 * time.Millisecond)

		err := repo.UpdateLastHit(ctx, "test-ns", "test-proj", "agent-1")

		assert.NoError(t, err)

		var updatedAgent model.Agent
		db.First(&updatedAgent, agent.ID)
		assert.True(t, updatedAgent.LastHitAt.After(originalLastHit))
	})

	t.Run("returns error when agent does not exist", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		err := repo.UpdateLastHit(ctx, "test-ns", "test-proj", "nonexistent")

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("only updates specified agent", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent1 := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		}
		agent2 := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-2",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		}
		db.Create(agent1)
		db.Create(agent2)

		var originalAgent2 model.Agent
		db.First(&originalAgent2, agent2.ID)
		originalLastHit2 := originalAgent2.LastHitAt

		time.Sleep(10 * time.Millisecond)

		err := repo.UpdateLastHit(ctx, "test-ns", "test-proj", "agent-1")

		assert.NoError(t, err)

		var updatedAgent2 model.Agent
		db.First(&updatedAgent2, agent2.ID)
		assert.Equal(t, originalLastHit2, updatedAgent2.LastHitAt)
	})

	t.Run("does not update other fields", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-1",
				Type:         commonTypes.AgentTypeTraefik,
				Status:       commonTypes.AgentStatusError,
				Error:        "some error",
				LoadDuration: commonTypes.NewDuration(100 * time.Millisecond),
				Version:      5,
			},
		}
		db.Create(agent)

		err := repo.UpdateLastHit(ctx, "test-ns", "test-proj", "agent-1")

		assert.NoError(t, err)

		var updatedAgent model.Agent
		db.First(&updatedAgent, agent.ID)
		assert.Equal(t, commonTypes.AgentStatusError, updatedAgent.Status)
		assert.Equal(t, "some error", updatedAgent.Error)
		assert.Equal(t, commonTypes.NewDuration(100*time.Millisecond), updatedAgent.LoadDuration)
		assert.Equal(t, 5, updatedAgent.Version)
		assert.Equal(t, commonTypes.AgentTypeTraefik, updatedAgent.Type)
	})
}

func TestAgentRepository_Delete(t *testing.T) {
	t.Run("delete existing agent", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		})

		err := repo.Delete(ctx, "test-ns", "test-proj", "agent-1")

		assert.NoError(t, err)

		var count int64
		db.Model(&model.Agent{}).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("delete nonexistent agent returns error", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		err := repo.Delete(ctx, "test-ns", "test-proj", "nonexistent")

		assert.Error(t, err)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("delete only specified agent", func(t *testing.T) {
		db := setupAgentTestDB(t)
		createTestAgentNamespace(t, db, "test-ns", "Test Namespace")
		createTestAgentProject(t, db, "test-ns", "test-proj", "Test Project")
		repo := NewAgentRepository(db)
		ctx := context.Background()

		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		})
		db.Create(&model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-2",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		})

		err := repo.Delete(ctx, "test-ns", "test-proj", "agent-1")

		assert.NoError(t, err)

		var count int64
		db.Model(&model.Agent{}).Count(&count)
		assert.Equal(t, int64(1), count)

		var remaining model.Agent
		db.First(&remaining)
		assert.Equal(t, "agent-2", remaining.Name)
	})
}
