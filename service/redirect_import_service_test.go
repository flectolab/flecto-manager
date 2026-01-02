package service

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	mockFlectoRepository "github.com/flectolab/flecto-manager/mocks/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/types"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRedirectImportServiceTest(t *testing.T) (*gomock.Controller, *mockFlectoRepository.MockRedirectDraftRepository, *gorm.DB, RedirectImportService) {
	ctrl := gomock.NewController(t)
	mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	err = db.AutoMigrate(&model.Namespace{}, &model.Project{}, &model.Redirect{}, &model.RedirectDraft{})
	assert.NoError(t, err)
	mockRepo.EXPECT().GetTx(gomock.Any()).Return(db).AnyTimes()
	svc := NewRedirectImportService(appContext.TestContext(nil), mockRepo)
	return ctrl, mockRepo, db, svc
}

func TestNewRedirectImportService(t *testing.T) {
	ctrl, mockRepo, _, svc := setupRedirectImportServiceTest(t)
	defer ctrl.Finish()

	assert.NotNil(t, svc)
	assert.NotNil(t, mockRepo)
}

func TestRedirectImportService_ValidateFile(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		size        int64
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid csv file",
			filename:    "redirects.csv",
			contentType: "text/csv",
			size:        1024,
			wantErr:     false,
		},
		{
			name:        "valid tsv file",
			filename:    "redirects.tsv",
			contentType: "text/tab-separated-values",
			size:        1024,
			wantErr:     false,
		},
		{
			name:        "valid with text/plain content type",
			filename:    "redirects.csv",
			contentType: "text/plain",
			size:        1024,
			wantErr:     false,
		},
		{
			name:        "valid with application/octet-stream content type",
			filename:    "redirects.tsv",
			contentType: "application/octet-stream",
			size:        1024,
			wantErr:     false,
		},
		{
			name:        "file too large",
			filename:    "redirects.csv",
			contentType: "text/csv",
			size:        3 * 1024 * 1024,
			wantErr:     true,
			errContains: "file too large",
		},
		{
			name:        "invalid extension txt",
			filename:    "redirects.txt",
			contentType: "text/plain",
			size:        1024,
			wantErr:     true,
			errContains: "invalid file type",
		},
		{
			name:        "invalid extension xlsx",
			filename:    "redirects.xlsx",
			contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			size:        1024,
			wantErr:     true,
			errContains: "invalid file type",
		},
		{
			name:        "invalid content type",
			filename:    "redirects.csv",
			contentType: "application/json",
			size:        1024,
			wantErr:     true,
			errContains: "invalid content type",
		},
		{
			name:        "uppercase extension",
			filename:    "redirects.CSV",
			contentType: "text/csv",
			size:        1024,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl, _, _, svc := setupRedirectImportServiceTest(t)
			defer ctrl.Finish()

			err := svc.ValidateFile(tt.filename, tt.contentType, tt.size)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedirectImportService_ParseFile(t *testing.T) {
	t.Run("success with valid data", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := "type\tsource\ttarget\tstatus\nBASIC\t/old\t/new\t301\nREGEX\t/pattern/(.*)\t/target/$1\tMOVED_PERMANENT"
		reader := strings.NewReader(input)

		rows, errs, err := svc.ParseFile(reader)

		assert.NoError(t, err)
		assert.Len(t, rows, 2)
		assert.Len(t, errs, 0)

		assert.Equal(t, 2, rows[0].LineNum)
		assert.Equal(t, commonTypes.RedirectTypeBasic, rows[0].Type)
		assert.Equal(t, "/old", rows[0].Source)
		assert.Equal(t, "/new", rows[0].Target)
		assert.Equal(t, commonTypes.RedirectStatusMovedPermanent, rows[0].Status)

		assert.Equal(t, 3, rows[1].LineNum)
		assert.Equal(t, commonTypes.RedirectTypeRegex, rows[1].Type)
	})

	t.Run("error invalid header column count", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := "type\tsource\ttarget\n"
		reader := strings.NewReader(input)

		rows, errs, err := svc.ParseFile(reader)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected 4 columns")
		assert.Nil(t, rows)
		assert.Nil(t, errs)
	})

	t.Run("error invalid header column name", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := "type\tsrc\ttarget\tstatus\n"
		reader := strings.NewReader(input)

		rows, errs, err := svc.ParseFile(reader)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "column 2 should be 'source'")
		assert.Nil(t, rows)
		assert.Nil(t, errs)
	})

	t.Run("error empty file", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := ""
		reader := strings.NewReader(input)

		_, _, err := svc.ParseFile(reader)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read header")
	})

	t.Run("parse error invalid type", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := "type\tsource\ttarget\tstatus\nINVALID_TYPE\t/old\t/new\t301"
		reader := strings.NewReader(input)

		rows, errs, err := svc.ParseFile(reader)

		assert.NoError(t, err)
		assert.Len(t, rows, 0)
		assert.Len(t, errs, 1)
		assert.Equal(t, ImportErrorInvalidType, errs[0].Reason)
	})

	t.Run("parse error invalid status", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := "type\tsource\ttarget\tstatus\nBASIC\t/old\t/new\tINVALID_STATUS"
		reader := strings.NewReader(input)

		rows, errs, err := svc.ParseFile(reader)

		assert.NoError(t, err)
		assert.Len(t, rows, 0)
		assert.Len(t, errs, 1)
		assert.Equal(t, ImportErrorInvalidStatus, errs[0].Reason)
	})

	t.Run("error duplicate source in file", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := "type\tsource\ttarget\tstatus\n" +
			"BASIC\t/same\t/target1\t301\n" +
			"BASIC\t/same\t/target2\t301"
		reader := strings.NewReader(input)

		rows, errs, err := svc.ParseFile(reader)

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Len(t, errs, 1)
		assert.Equal(t, ImportErrorDuplicateInFile, errs[0].Reason)
		assert.Contains(t, errs[0].Message, "line 2")
	})

	t.Run("error wrong column count in row", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := "type\tsource\ttarget\tstatus\nBASIC\t/old\t/new"
		reader := strings.NewReader(input)

		rows, errs, err := svc.ParseFile(reader)

		assert.NoError(t, err)
		assert.Len(t, rows, 0)
		assert.Len(t, errs, 1)
		assert.Equal(t, ImportErrorInvalidFormat, errs[0].Reason)
		assert.Contains(t, errs[0].Message, "expected 4 columns")
	})

	t.Run("all redirect types", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := "type\tsource\ttarget\tstatus\n" +
			"BASIC\t/a\t/b\t301\n" +
			"BASIC_HOST\t/c\t/d\t301\n" +
			"REGEX\t/e\t/f\t301\n" +
			"REGEX_HOST\t/g\t/h\t301"
		reader := strings.NewReader(input)

		rows, errs, err := svc.ParseFile(reader)

		assert.NoError(t, err)
		assert.Len(t, rows, 4)
		assert.Len(t, errs, 0)

		assert.Equal(t, commonTypes.RedirectTypeBasic, rows[0].Type)
		assert.Equal(t, commonTypes.RedirectTypeBasicHost, rows[1].Type)
		assert.Equal(t, commonTypes.RedirectTypeRegex, rows[2].Type)
		assert.Equal(t, commonTypes.RedirectTypeRegexHost, rows[3].Type)
	})

	t.Run("all redirect statuses", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := "type\tsource\ttarget\tstatus\n" +
			"BASIC\t/a\t/b\t301\n" +
			"BASIC\t/c\t/d\t302\n" +
			"BASIC\t/e\t/f\t307\n" +
			"BASIC\t/g\t/h\t308\n" +
			"BASIC\t/i\t/j\tMOVED_PERMANENT\n" +
			"BASIC\t/k\t/l\tFOUND\n" +
			"BASIC\t/m\t/n\tTEMPORARY_REDIRECT\n" +
			"BASIC\t/o\t/p\tPERMANENT_REDIRECT"
		reader := strings.NewReader(input)

		rows, errs, err := svc.ParseFile(reader)

		assert.NoError(t, err)
		assert.Len(t, rows, 8)
		assert.Len(t, errs, 0)

		assert.Equal(t, commonTypes.RedirectStatusMovedPermanent, rows[0].Status)
		assert.Equal(t, commonTypes.RedirectStatusFound, rows[1].Status)
		assert.Equal(t, commonTypes.RedirectStatusTemporary, rows[2].Status)
		assert.Equal(t, commonTypes.RedirectStatusPermanent, rows[3].Status)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		input := "type\tsource\ttarget\tstatus\n  BASIC  \t  /old  \t  /new  \t  301  "
		reader := strings.NewReader(input)

		rows, errs, err := svc.ParseFile(reader)

		assert.NoError(t, err)
		assert.Len(t, rows, 1)
		assert.Len(t, errs, 0)
		assert.Equal(t, "/old", rows[0].Source)
		assert.Equal(t, "/new", rows[0].Target)
	})

	t.Run("parse error empty source", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		// Build TSV data with empty source field using explicit bytes
		// Header: type<tab>source<tab>target<tab>status<newline>
		// Data:   BASIC<tab><empty><tab>/new<tab>301
		data := []byte("type\tsource\ttarget\tstatus\nBASIC\t\t/new\t301\n")
		reader := bytes.NewReader(data)

		rows, errs, err := svc.ParseFile(reader)

		assert.NoError(t, err)
		assert.Len(t, rows, 0)
		assert.Len(t, errs, 1)
		assert.Equal(t, ImportErrorEmptySource, errs[0].Reason)
	})

	t.Run("parse error empty target", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		// Build TSV data with empty target field using explicit bytes
		// Header: type<tab>source<tab>target<tab>status<newline>
		// Data:   BASIC<tab>/old<tab><empty><tab>301
		data := []byte("type\tsource\ttarget\tstatus\nBASIC\t/old\t\t301\n")
		reader := bytes.NewReader(data)

		rows, errs, err := svc.ParseFile(reader)

		assert.NoError(t, err)
		assert.Len(t, rows, 0)
		assert.Len(t, errs, 1)
		assert.Equal(t, ImportErrorEmptyTarget, errs[0].Reason)
	})
}

func TestRedirectImportService_Import(t *testing.T) {
	t.Run("success create new redirects", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/old1", Target: "/new1", Status: commonTypes.RedirectStatusMovedPermanent},
			{LineNum: 3, Type: commonTypes.RedirectTypeBasic, Source: "/old2", Target: "/new2", Status: commonTypes.RedirectStatusFound},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/old1", nil, nil).Return(true, nil)
		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/old2", nil, nil).Return(true, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: false})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 2, result.TotalLines)
		assert.Equal(t, 2, result.ImportedCount)
		assert.Equal(t, 0, result.SkippedCount)
		assert.Equal(t, 0, result.ErrorCount)

		// Verify redirects and drafts were created
		var redirects []model.Redirect
		db.Find(&redirects)
		assert.Len(t, redirects, 2)

		var drafts []model.RedirectDraft
		db.Find(&drafts)
		assert.Len(t, drafts, 2)
	})

	t.Run("success with empty rows", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		result, err := svc.Import(ctx, "ns", "proj", []ParsedRedirectRow{}, ImportRedirectOptions{})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 0, result.TotalLines)
		assert.Equal(t, 0, result.ImportedCount)
	})

	t.Run("invalid data", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasicHost, Source: "/old1", Target: "/new1", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/old1", nil, nil).Return(true, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: false})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, 1, result.TotalLines)
		assert.Equal(t, 0, result.ImportedCount)
		assert.Equal(t, 0, result.SkippedCount)
		assert.Equal(t, 1, result.ErrorCount)

	})

	t.Run("error source already exists without overwrite", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/existing", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/existing", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: false})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, 1, result.ErrorCount)
		assert.Equal(t, ImportErrorSourceAlreadyExists, result.Errors[0].Reason)
	})

	t.Run("success overwrite existing draft", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create existing redirect with draft
		redirect := &model.Redirect{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Redirect: &commonTypes.Redirect{
				Source: "/existing",
				Target: "/old-target",
				Type:   commonTypes.RedirectTypeBasic,
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			IsPublished: types.Ptr(true),
		}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
			NewRedirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/existing",
				Target: "/draft-target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(draft)

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/existing", Target: "/imported-target", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/existing", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1, result.ImportedCount)

		// Verify draft was updated
		var updatedDraft model.RedirectDraft
		db.First(&updatedDraft, draft.ID)
		assert.Equal(t, "/imported-target", updatedDraft.NewRedirect.Target)
	})

	t.Run("skip when data is identical", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create existing redirect with draft containing same data
		redirect := &model.Redirect{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Redirect: &commonTypes.Redirect{
				Source: "/existing",
				Target: "/target",
				Type:   commonTypes.RedirectTypeBasic,
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			IsPublished: types.Ptr(true),
		}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
			NewRedirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/existing",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(draft)

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/existing", Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/existing", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 0, result.ImportedCount)
		assert.Equal(t, 1, result.SkippedCount)
	})

	t.Run("skip when published data is identical", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create existing published redirect without draft
		redirect := &model.Redirect{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Redirect: &commonTypes.Redirect{
				Source: "/existing",
				Target: "/target",
				Type:   commonTypes.RedirectTypeBasic,
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			IsPublished: types.Ptr(true),
		}
		db.Create(redirect)

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/existing", Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/existing", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 0, result.ImportedCount)
		assert.Equal(t, 1, result.SkippedCount)
	})

	t.Run("create draft for published redirect with different data", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create existing published redirect without draft
		redirect := &model.Redirect{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Redirect: &commonTypes.Redirect{
				Source: "/existing",
				Target: "/old-target",
				Type:   commonTypes.RedirectTypeBasic,
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			IsPublished: types.Ptr(true),
		}
		db.Create(redirect)

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/existing", Target: "/new-target", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/existing", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1, result.ImportedCount)

		// Verify draft was created
		var drafts []model.RedirectDraft
		db.Where("old_redirect_id = ?", redirect.ID).Find(&drafts)
		assert.Len(t, drafts, 1)
		assert.Equal(t, "/new-target", drafts[0].NewRedirect.Target)
		assert.Equal(t, model.DraftChangeTypeUpdate, drafts[0].ChangeType)
	})

	t.Run("update existing unpublished draft", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		// Create unpublished redirect with draft
		redirect := &model.Redirect{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			IsPublished:   types.Ptr(false),
		}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeCreate,
			NewRedirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/new-source",
				Target: "/old-target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(draft)

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/new-source", Target: "/updated-target", Status: commonTypes.RedirectStatusFound},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/new-source", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1, result.ImportedCount)

		// Verify draft was updated
		var updatedDraft model.RedirectDraft
		db.First(&updatedDraft, draft.ID)
		assert.Equal(t, "/updated-target", updatedDraft.NewRedirect.Target)
		assert.Equal(t, commonTypes.RedirectStatusFound, updatedDraft.NewRedirect.Status)
	})

	t.Run("error checking source availability", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/source", Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/source", nil, nil).Return(false, errors.New("database error"))

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check source availability")
		assert.Nil(t, result)
	})

	t.Run("all rows filtered out by errors", func(t *testing.T) {
		ctrl, mockRepo, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()
		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/existing", Target: "/new", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/existing", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: false})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, 1, result.ErrorCount)
		assert.Equal(t, 0, result.ImportedCount)
	})

	t.Run("error saving existing draft update", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		redirect := &model.Redirect{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Redirect: &commonTypes.Redirect{
				Source: "/existing",
				Target: "/old-target",
				Type:   commonTypes.RedirectTypeBasic,
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			IsPublished: types.Ptr(true),
		}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeUpdate,
			NewRedirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/existing",
				Target: "/draft-target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(draft)

		db.Migrator().DropTable(&model.RedirectDraft{})

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/existing", Target: "/new-target", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/existing", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.Equal(t, 1, result.ErrorCount)
		assert.Equal(t, ImportErrorDatabaseError, result.Errors[0].Reason)
	})

	t.Run("error creating redirect in createNewDraft", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		db.Migrator().DropTable(&model.Redirect{})

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/new", Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/new", nil, nil).Return(true, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: false})

		assert.NoError(t, err)
		assert.Equal(t, 1, result.ErrorCount)
		assert.Equal(t, ImportErrorDatabaseError, result.Errors[0].Reason)
		assert.Contains(t, result.Errors[0].Message, "failed to create redirect")
	})

	t.Run("error creating draft in createNewDraft", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		db.Migrator().DropTable(&model.RedirectDraft{})

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/new", Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/new", nil, nil).Return(true, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: false})

		assert.NoError(t, err)
		assert.Equal(t, 1, result.ErrorCount)
		assert.Equal(t, ImportErrorDatabaseError, result.Errors[0].Reason)
		assert.Contains(t, result.Errors[0].Message, "failed to create redirect draft")
	})

	t.Run("error creating draft for published redirect", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		redirect := &model.Redirect{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			Redirect: &commonTypes.Redirect{
				Source: "/existing",
				Target: "/old-target",
				Type:   commonTypes.RedirectTypeBasic,
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			IsPublished: types.Ptr(true),
		}
		db.Create(redirect)

		db.Migrator().DropTable(&model.RedirectDraft{})

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/existing", Target: "/new-target", Status: commonTypes.RedirectStatusFound},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/existing", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.Equal(t, 1, result.ErrorCount)
		assert.Equal(t, ImportErrorDatabaseError, result.Errors[0].Reason)
		assert.Contains(t, result.Errors[0].Message, "no such table: redirect_drafts")
	})

	t.Run("error updating unpublished draft", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		redirect := &model.Redirect{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			IsPublished:   types.Ptr(false),
		}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeCreate,
			NewRedirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/new-source",
				Target: "/old-target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(draft)

		db.Migrator().DropTable(&model.RedirectDraft{})

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/new-source", Target: "/updated-target", Status: commonTypes.RedirectStatusFound},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/new-source", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.Equal(t, 1, result.ErrorCount)
		assert.Equal(t, ImportErrorDatabaseError, result.Errors[0].Reason)
	})

	t.Run("skip when unpublished draft data is identical", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		redirect := &model.Redirect{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			IsPublished:   types.Ptr(false),
		}
		db.Create(redirect)

		draft := &model.RedirectDraft{
			NamespaceCode: "ns",
			ProjectCode:   "proj",
			OldRedirectID: &redirect.ID,
			ChangeType:    model.DraftChangeTypeCreate,
			NewRedirect: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/new-source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(draft)

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/new-source", Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/new-source", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 0, result.ImportedCount)
		assert.Equal(t, 1, result.SkippedCount)
	})

	t.Run("fallthrough to createNewDraft when source unavailable but not found", func(t *testing.T) {
		ctrl, mockRepo, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		ctx := context.Background()

		rows := []ParsedRedirectRow{
			{LineNum: 2, Type: commonTypes.RedirectTypeBasic, Source: "/ghost", Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent},
		}

		mockRepo.EXPECT().CheckSourceAvailability(ctx, "ns", "proj", "/ghost", nil, nil).Return(false, nil)

		result, err := svc.Import(ctx, "ns", "proj", rows, ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1, result.ImportedCount)

		var redirects []model.Redirect
		db.Find(&redirects)
		assert.Len(t, redirects, 1)
	})
}

func TestParseRedirectType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    commonTypes.RedirectType
		wantErr bool
	}{
		{"basic lowercase", "basic", commonTypes.RedirectTypeBasic, false},
		{"basic uppercase", "BASIC", commonTypes.RedirectTypeBasic, false},
		{"basic mixed", "Basic", commonTypes.RedirectTypeBasic, false},
		{"basic_host", "BASIC_HOST", commonTypes.RedirectTypeBasicHost, false},
		{"regex", "REGEX", commonTypes.RedirectTypeRegex, false},
		{"regex_host", "REGEX_HOST", commonTypes.RedirectTypeRegexHost, false},
		{"invalid", "INVALID", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRedirectType(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseRedirectStatus(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    commonTypes.RedirectStatus
		wantErr bool
	}{
		{"301", "301", commonTypes.RedirectStatusMovedPermanent, false},
		{"302", "302", commonTypes.RedirectStatusFound, false},
		{"307", "307", commonTypes.RedirectStatusTemporary, false},
		{"308", "308", commonTypes.RedirectStatusPermanent, false},
		{"moved_permanent", "MOVED_PERMANENT", commonTypes.RedirectStatusMovedPermanent, false},
		{"found", "FOUND", commonTypes.RedirectStatusFound, false},
		{"temporary_redirect", "TEMPORARY_REDIRECT", commonTypes.RedirectStatusTemporary, false},
		{"permanent_redirect", "PERMANENT_REDIRECT", commonTypes.RedirectStatusPermanent, false},
		{"lowercase", "moved_permanent", commonTypes.RedirectStatusMovedPermanent, false},
		{"invalid", "INVALID", "", true},
		{"invalid number", "300", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRedirectStatus(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestRedirectsAreEqual(t *testing.T) {
	tests := []struct {
		name string
		a    *commonTypes.Redirect
		b    *commonTypes.Redirect
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil",
			a:    nil,
			b:    &commonTypes.Redirect{},
			want: false,
		},
		{
			name: "b nil",
			a:    &commonTypes.Redirect{},
			b:    nil,
			want: false,
		},
		{
			name: "equal",
			a: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			b: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			want: true,
		},
		{
			name: "different type",
			a: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			b: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeRegex,
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			want: false,
		},
		{
			name: "different source",
			a: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source1",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			b: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source2",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			want: false,
		},
		{
			name: "different target",
			a: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target1",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			b: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target2",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			want: false,
		},
		{
			name: "different status",
			a: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusMovedPermanent,
			},
			b: &commonTypes.Redirect{
				Type:   commonTypes.RedirectTypeBasic,
				Source: "/source",
				Target: "/target",
				Status: commonTypes.RedirectStatusFound,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redirectsAreEqual(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedirectImportService_GetTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
	svc := NewRedirectImportService(appContext.TestContext(nil), mockRepo)

	ctx := context.Background()
	mockRepo.EXPECT().GetTx(ctx).Return(nil)

	result := svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestRedirectImportService_GetQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
	svc := NewRedirectImportService(appContext.TestContext(nil), mockRepo)

	ctx := context.Background()
	mockRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := svc.GetQuery(ctx)
	assert.Nil(t, result)
}
