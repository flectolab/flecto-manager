package service

import (
	"context"
	"errors"
	"testing"
	"time"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func setupAgentServiceTest(t *testing.T) (*gomock.Controller, *mockFlectoRepository.MockAgentRepository, AgentService) {
	ctrl := gomock.NewController(t)
	mockAgentRepo := mockFlectoRepository.NewMockAgentRepository(ctrl)
	svc := NewAgentService(appContext.TestContext(nil), mockAgentRepo)
	return ctrl, mockAgentRepo, svc
}

func TestNewAgentService(t *testing.T) {
	ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mockAgentRepo)
}

func TestAgentService_Upsert(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:         "agent-1",
				Type:         commonTypes.AgentTypeTraefik,
				Status:       commonTypes.AgentStatusSuccess,
				Version:      1,
				LoadDuration: commonTypes.NewDuration(100 * time.Millisecond),
			},
		}

		mockAgentRepo.EXPECT().
			Upsert(ctx, agent).
			Return(nil)

		err := svc.Upsert(ctx, agent)

		assert.NoError(t, err)
	})

	t.Run("validation error invalid type", func(t *testing.T) {
		ctrl, _, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentType("invalid"),
				Status: commonTypes.AgentStatusSuccess,
			},
		}

		err := svc.Upsert(ctx, agent)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid agent type")
	})

	t.Run("validation error invalid name", func(t *testing.T) {
		ctrl, _, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent with spaces",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		}

		err := svc.Upsert(ctx, agent)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid agent name")
	})

	t.Run("validation error invalid status", func(t *testing.T) {
		ctrl, _, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatus("invalid"),
			},
		}

		err := svc.Upsert(ctx, agent)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid agent status")
	})

	t.Run("repository error", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		agent := &model.Agent{
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:    "agent-1",
				Type:    commonTypes.AgentTypeTraefik,
				Status:  commonTypes.AgentStatusSuccess,
				Version: 1,
			},
		}
		expectedErr := errors.New("database error")

		mockAgentRepo.EXPECT().
			Upsert(ctx, agent).
			Return(expectedErr)

		err := svc.Upsert(ctx, agent)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestAgentService_GetByName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedAgent := &model.Agent{
			ID:            1,
			NamespaceCode: "test-ns",
			ProjectCode:   "test-proj",
			Agent: commonTypes.Agent{
				Name:   "agent-1",
				Type:   commonTypes.AgentTypeTraefik,
				Status: commonTypes.AgentStatusSuccess,
			},
		}

		mockAgentRepo.EXPECT().
			FindByName(ctx, "test-ns", "test-proj", "agent-1").
			Return(expectedAgent, nil)

		result, err := svc.GetByName(ctx, "test-ns", "test-proj", "agent-1")

		assert.NoError(t, err)
		assert.Equal(t, expectedAgent, result)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		mockAgentRepo.EXPECT().
			FindByName(ctx, "test-ns", "test-proj", "nonexistent").
			Return(nil, expectedErr)

		result, err := svc.GetByName(ctx, "test-ns", "test-proj", "nonexistent")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestAgentService_FindByProject(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedAgents := []model.Agent{
			{
				ID:            1,
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				Agent: commonTypes.Agent{
					Name:   "agent-1",
					Type:   commonTypes.AgentTypeTraefik,
					Status: commonTypes.AgentStatusSuccess,
				},
			},
			{
				ID:            2,
				NamespaceCode: "test-ns",
				ProjectCode:   "test-proj",
				Agent: commonTypes.Agent{
					Name:   "agent-2",
					Type:   commonTypes.AgentTypeTraefik,
					Status: commonTypes.AgentStatusError,
				},
			},
		}

		mockAgentRepo.EXPECT().
			FindByProject(ctx, "test-ns", "test-proj").
			Return(expectedAgents, nil)

		result, err := svc.FindByProject(ctx, "test-ns", "test-proj")

		assert.NoError(t, err)
		assert.Equal(t, expectedAgents, result)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("database error")

		mockAgentRepo.EXPECT().
			FindByProject(ctx, "test-ns", "test-proj").
			Return(nil, expectedErr)

		result, err := svc.FindByProject(ctx, "test-ns", "test-proj")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestAgentService_SearchPaginate(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		limit := 10
		offset := 5
		pagination := &commonTypes.PaginationInput{
			Limit:  &limit,
			Offset: &offset,
		}
		expectedAgents := []model.Agent{
			{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
			{ID: 2, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
		}

		mockAgentRepo.EXPECT().
			SearchPaginate(ctx, nil, 10, 5).
			Return(expectedAgents, int64(50), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 50, result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 5, result.Offset)
		assert.Len(t, result.Items, 2)
	})

	t.Run("success with default pagination", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &commonTypes.PaginationInput{}
		expectedAgents := []model.Agent{
			{ID: 1, NamespaceCode: "test-ns", ProjectCode: "test-proj"},
		}

		mockAgentRepo.EXPECT().
			SearchPaginate(ctx, nil, commonTypes.DefaultLimit, commonTypes.DefaultOffset).
			Return(expectedAgents, int64(1), nil)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, commonTypes.DefaultLimit, result.Limit)
		assert.Equal(t, commonTypes.DefaultOffset, result.Offset)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		pagination := &commonTypes.PaginationInput{}
		expectedErr := errors.New("search error")

		mockAgentRepo.EXPECT().
			SearchPaginate(ctx, nil, commonTypes.DefaultLimit, commonTypes.DefaultOffset).
			Return(nil, int64(0), expectedErr)

		result, err := svc.SearchPaginate(ctx, pagination, nil)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, result)
	})
}

func TestAgentService_CountByProjectAndStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		before := time.Now()

		mockAgentRepo.EXPECT().
			CountByProjectAndStatus(ctx, "test-ns", "test-proj", commonTypes.AgentStatusError, before).
			Return(int64(5), nil)

		count, err := svc.CountByProjectAndStatus(ctx, "test-ns", "test-proj", commonTypes.AgentStatusError, before)

		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		before := time.Now()
		expectedErr := errors.New("database error")

		mockAgentRepo.EXPECT().
			CountByProjectAndStatus(ctx, "test-ns", "test-proj", commonTypes.AgentStatusError, before).
			Return(int64(0), expectedErr)

		count, err := svc.CountByProjectAndStatus(ctx, "test-ns", "test-proj", commonTypes.AgentStatusError, before)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestAgentService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockAgentRepo.EXPECT().
			Delete(ctx, "test-ns", "test-proj", "agent-1").
			Return(nil)

		err := svc.Delete(ctx, "test-ns", "test-proj", "agent-1")

		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("record not found")

		mockAgentRepo.EXPECT().
			Delete(ctx, "test-ns", "test-proj", "nonexistent").
			Return(expectedErr)

		err := svc.Delete(ctx, "test-ns", "test-proj", "nonexistent")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestAgentService_GetTx(t *testing.T) {
	ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockAgentRepo.EXPECT().GetTx(ctx).Return(nil)

	result := svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestAgentService_GetQuery(t *testing.T) {
	ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockAgentRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := svc.GetQuery(ctx)
	assert.Nil(t, result)
}

func TestAgentService_UpdateLastHit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		mockAgentRepo.EXPECT().
			UpdateLastHit(ctx, "test-ns", "test-proj", "agent-1").
			Return(nil)

		err := svc.UpdateLastHit(ctx, "test-ns", "test-proj", "agent-1")

		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		ctrl, mockAgentRepo, svc := setupAgentServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		expectedErr := errors.New("update error")

		mockAgentRepo.EXPECT().
			UpdateLastHit(ctx, "test-ns", "test-proj", "agent-1").
			Return(expectedErr)

		err := svc.UpdateLastHit(ctx, "test-ns", "test-proj", "agent-1")

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}
