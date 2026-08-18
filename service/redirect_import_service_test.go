package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	svc := NewRedirectImportService(appContext.TestContext(nil), mockRepo, newTestActivityService(t, db))
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
			name:        "file at the size limit",
			filename:    "redirects.csv",
			contentType: "text/csv",
			size:        MaxImportFileSize,
			wantErr:     false,
		},
		{
			name:        "file too large",
			filename:    "redirects.csv",
			contentType: "text/csv",
			size:        MaxImportFileSize + 1,
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

// tsvFile builds an import file from "type\tsource\ttarget\tstatus" rows.
func tsvFile(rows ...string) io.Reader {
	return strings.NewReader("type\tsource\ttarget\tstatus\n" + strings.Join(rows, "\n") + "\n")
}

func TestRedirectImportService_Import(t *testing.T) {
	t.Run("success create new redirects", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/old1\t/new1\t301",
			"BASIC\t/old2\t/new2\t302",
		), ImportRedirectOptions{Overwrite: false})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 2, result.TotalLines)
		assert.Equal(t, 2, result.ImportedCount)
		assert.Equal(t, 0, result.SkippedCount)
		assert.Equal(t, 0, result.ErrorCount)

		var redirects []model.Redirect
		db.Find(&redirects)
		assert.Len(t, redirects, 2)

		var drafts []model.RedirectDraft
		db.Find(&drafts)
		assert.Len(t, drafts, 2)
		// every draft points at the redirect created for it
		for _, d := range drafts {
			assert.Equal(t, model.DraftChangeTypeCreate, d.ChangeType)
			assert.NotNil(t, d.OldRedirectID)
		}
	})

	t.Run("draft is linked to its own redirect", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		_, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/a\t/target-a\t301",
			"BASIC\t/b\t/target-b\t301",
			"BASIC\t/c\t/target-c\t301",
		), ImportRedirectOptions{})
		assert.NoError(t, err)

		var drafts []model.RedirectDraft
		db.Order("new_source").Find(&drafts)
		assert.Len(t, drafts, 3)

		seen := map[int64]bool{}
		for _, d := range drafts {
			assert.False(t, seen[*d.OldRedirectID], "two drafts share the same redirect")
			seen[*d.OldRedirectID] = true
		}
	})

	t.Run("success with header only", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		result, err := svc.Import(context.Background(), "ns", "proj",
			strings.NewReader("type\tsource\ttarget\tstatus\n"), ImportRedirectOptions{})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 0, result.TotalLines)
		assert.Equal(t, 0, result.ImportedCount)
	})

	t.Run("invalid header", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		result, err := svc.Import(context.Background(), "ns", "proj",
			strings.NewReader("a\tb\tc\td\n"), ImportRedirectOptions{})

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("parse errors are reported by Import", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/ok\t/target\t301",
			"NOPE\t/bad-type\t/target\t301",
			"BASIC\t\t/target\t301",
			"BASIC\t/no-target\t\t301",
			"BASIC\t/bad-status\t/target\t999",
			"BASIC\t/ok\t/other\t301",
		), ImportRedirectOptions{})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, 6, result.TotalLines)
		assert.Equal(t, 1, result.ImportedCount)
		assert.Equal(t, 5, result.ErrorCount)

		reasons := make([]ImportErrorReason, 0, len(result.Errors))
		for _, e := range result.Errors {
			reasons = append(reasons, e.Reason)
		}
		assert.Equal(t, []ImportErrorReason{
			ImportErrorInvalidType,
			ImportErrorEmptySource,
			ImportErrorEmptyTarget,
			ImportErrorInvalidStatus,
			ImportErrorDuplicateInFile,
		}, reasons)
	})

	t.Run("invalid redirect data", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		// BASIC_HOST requires a host in the source
		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC_HOST\t/old1\t/new1\t301",
		), ImportRedirectOptions{Overwrite: false})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, 1, result.TotalLines)
		assert.Equal(t, 0, result.ImportedCount)
		assert.Equal(t, 1, result.ErrorCount)
		assert.Equal(t, ImportErrorInvalidRedirect, result.Errors[0].Reason)
	})

	t.Run("error source already exists without overwrite", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		db.Create(&model.Redirect{
			NamespaceCode: "ns", ProjectCode: "proj", IsPublished: types.Ptr(true),
			Redirect: &commonTypes.Redirect{
				Type: commonTypes.RedirectTypeBasic, Source: "/existing",
				Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent,
			},
		})

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/existing\t/new\t301",
		), ImportRedirectOptions{Overwrite: false})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, 1, result.ErrorCount)
		assert.Equal(t, 0, result.ImportedCount)
		assert.Equal(t, ImportErrorSourceAlreadyExists, result.Errors[0].Reason)
	})

	t.Run("success overwrite existing draft", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		redirect := &model.Redirect{
			NamespaceCode: "ns", ProjectCode: "proj", IsPublished: types.Ptr(true),
			Redirect: &commonTypes.Redirect{
				Type: commonTypes.RedirectTypeBasic, Source: "/existing",
				Target: "/old-target", Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(redirect)
		draft := &model.RedirectDraft{
			NamespaceCode: "ns", ProjectCode: "proj",
			OldRedirectID: &redirect.ID, ChangeType: model.DraftChangeTypeUpdate,
			NewRedirect: &commonTypes.Redirect{
				Type: commonTypes.RedirectTypeBasic, Source: "/existing",
				Target: "/draft-target", Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(draft)

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/existing\t/imported-target\t301",
		), ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1, result.ImportedCount)

		var updated model.RedirectDraft
		db.First(&updated, draft.ID)
		assert.Equal(t, "/imported-target", updated.NewRedirect.Target)

		// the upsert must not have inserted a second draft
		var count int64
		db.Model(&model.RedirectDraft{}).Count(&count)
		assert.Equal(t, int64(1), count)
	})

	t.Run("skip when draft data is identical", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		redirect := &model.Redirect{
			NamespaceCode: "ns", ProjectCode: "proj", IsPublished: types.Ptr(true),
			Redirect: &commonTypes.Redirect{
				Type: commonTypes.RedirectTypeBasic, Source: "/existing",
				Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(redirect)
		db.Create(&model.RedirectDraft{
			NamespaceCode: "ns", ProjectCode: "proj",
			OldRedirectID: &redirect.ID, ChangeType: model.DraftChangeTypeUpdate,
			NewRedirect: &commonTypes.Redirect{
				Type: commonTypes.RedirectTypeBasic, Source: "/existing",
				Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent,
			},
		})

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/existing\t/target\t301",
		), ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 0, result.ImportedCount)
		assert.Equal(t, 1, result.SkippedCount)
	})

	t.Run("skip when published data is identical", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		db.Create(&model.Redirect{
			NamespaceCode: "ns", ProjectCode: "proj", IsPublished: types.Ptr(true),
			Redirect: &commonTypes.Redirect{
				Type: commonTypes.RedirectTypeBasic, Source: "/existing",
				Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent,
			},
		})

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/existing\t/target\t301",
		), ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 0, result.ImportedCount)
		assert.Equal(t, 1, result.SkippedCount)
	})

	t.Run("create draft for published redirect with different data", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		redirect := &model.Redirect{
			NamespaceCode: "ns", ProjectCode: "proj", IsPublished: types.Ptr(true),
			Redirect: &commonTypes.Redirect{
				Type: commonTypes.RedirectTypeBasic, Source: "/existing",
				Target: "/old-target", Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(redirect)

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/existing\t/new-target\t301",
		), ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1, result.ImportedCount)

		var drafts []model.RedirectDraft
		db.Where("old_redirect_id = ?", redirect.ID).Find(&drafts)
		assert.Len(t, drafts, 1)
		assert.Equal(t, "/new-target", drafts[0].NewRedirect.Target)
		assert.Equal(t, model.DraftChangeTypeUpdate, drafts[0].ChangeType)
	})

	t.Run("update existing unpublished draft", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		redirect := &model.Redirect{NamespaceCode: "ns", ProjectCode: "proj", IsPublished: types.Ptr(false)}
		db.Create(redirect)
		draft := &model.RedirectDraft{
			NamespaceCode: "ns", ProjectCode: "proj",
			OldRedirectID: &redirect.ID, ChangeType: model.DraftChangeTypeCreate,
			NewRedirect: &commonTypes.Redirect{
				Type: commonTypes.RedirectTypeBasic, Source: "/new-source",
				Target: "/old-target", Status: commonTypes.RedirectStatusMovedPermanent,
			},
		}
		db.Create(draft)

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/new-source\t/updated-target\t302",
		), ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1, result.ImportedCount)

		var updated model.RedirectDraft
		db.First(&updated, draft.ID)
		assert.Equal(t, "/updated-target", updated.NewRedirect.Target)
		assert.Equal(t, commonTypes.RedirectStatusFound, updated.NewRedirect.Status)
		assert.Equal(t, model.DraftChangeTypeCreate, updated.ChangeType, "change type must be preserved")
	})

	t.Run("skip when unpublished draft data is identical", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		redirect := &model.Redirect{NamespaceCode: "ns", ProjectCode: "proj", IsPublished: types.Ptr(false)}
		db.Create(redirect)
		db.Create(&model.RedirectDraft{
			NamespaceCode: "ns", ProjectCode: "proj",
			OldRedirectID: &redirect.ID, ChangeType: model.DraftChangeTypeCreate,
			NewRedirect: &commonTypes.Redirect{
				Type: commonTypes.RedirectTypeBasic, Source: "/new-source",
				Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent,
			},
		})

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/new-source\t/target\t301",
		), ImportRedirectOptions{Overwrite: true})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 0, result.ImportedCount)
		assert.Equal(t, 1, result.SkippedCount)
	})

	t.Run("a DELETE draft does not block the source", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		redirect := &model.Redirect{NamespaceCode: "ns", ProjectCode: "proj", IsPublished: types.Ptr(false)}
		db.Create(redirect)
		db.Create(&model.RedirectDraft{
			NamespaceCode: "ns", ProjectCode: "proj",
			OldRedirectID: &redirect.ID, ChangeType: model.DraftChangeTypeDelete,
			NewRedirect: &commonTypes.Redirect{
				Type: commonTypes.RedirectTypeBasic, Source: "/gone",
				Target: "/target", Status: commonTypes.RedirectStatusMovedPermanent,
			},
		})

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/gone\t/target\t301",
		), ImportRedirectOptions{Overwrite: false})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, 1, result.ImportedCount)
	})

	t.Run("rows spanning several chunks are all imported", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		total := importChunkSize + 250
		rows := make([]string, 0, total)
		for i := 0; i < total; i++ {
			rows = append(rows, fmt.Sprintf("BASIC\t/src-%d\t/target-%d\t301", i, i))
		}

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(rows...), ImportRedirectOptions{})

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, total, result.TotalLines)
		assert.Equal(t, total, result.ImportedCount)

		var count int64
		db.Model(&model.RedirectDraft{}).Count(&count)
		assert.Equal(t, int64(total), count)

		// each draft must still point at a distinct redirect
		var linked int64
		db.Model(&model.RedirectDraft{}).Distinct("old_redirect_id").Count(&linked)
		assert.Equal(t, int64(total), linked)
	})

	t.Run("reported errors are capped but the count stays exact", func(t *testing.T) {
		ctrl, _, _, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		total := maxReportedErrors + 42
		rows := make([]string, 0, total)
		for i := 0; i < total; i++ {
			rows = append(rows, fmt.Sprintf("NOPE\t/src-%d\t/target-%d\t301", i, i))
		}

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(rows...), ImportRedirectOptions{})

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, total, result.ErrorCount)
		assert.Len(t, result.Errors, maxReportedErrors)
	})

	t.Run("database failure aborts the import", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		db.Migrator().DropTable(&model.RedirectDraft{})

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/new\t/target\t301",
		), ImportRedirectOptions{})

		assert.Error(t, err)
		assert.Nil(t, result)

		// the transaction rolled back: no orphan redirect left behind
		var count int64
		db.Model(&model.Redirect{}).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("database failure on redirect insert aborts the import", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		db.Migrator().DropTable(&model.Redirect{})

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(
			"BASIC\t/new\t/target\t301",
		), ImportRedirectOptions{})

		assert.Error(t, err)
		assert.Nil(t, result)
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
	svc := NewRedirectImportService(appContext.TestContext(nil), mockRepo, newTestActivityService(t, nil))

	ctx := context.Background()
	mockRepo.EXPECT().GetTx(ctx).Return(nil)

	result := svc.GetTx(ctx)
	assert.Nil(t, result)
}

func TestRedirectImportService_GetQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockFlectoRepository.NewMockRedirectDraftRepository(ctrl)
	svc := NewRedirectImportService(appContext.TestContext(nil), mockRepo, newTestActivityService(t, nil))

	ctx := context.Background()
	mockRepo.EXPECT().GetQuery(ctx).Return(nil)

	result := svc.GetQuery(ctx)
	assert.Nil(t, result)
}

func TestRedirectImportService_ActivityEvents(t *testing.T) {
	t.Run("an import records exactly one aggregated event", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		rows := make([]string, 0, 50)
		for i := 0; i < 50; i++ {
			rows = append(rows, fmt.Sprintf("BASIC\t/old%d\t/new%d\t301", i, i))
		}

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(rows...),
			ImportRedirectOptions{Overwrite: true, Filename: "redirects.tsv"})
		assert.NoError(t, err)
		assert.Equal(t, 50, result.ImportedCount)

		// 50 imported redirects, still a single journal entry
		assert.Equal(t, int64(1), countActivityEvents(t, db))

		event := lastActivityEvent(t, db)
		assert.Equal(t, model.ActivityResourceRedirect, event.Resource)
		assert.Equal(t, model.ActivityActionImport, event.Action)
		assert.Nil(t, event.ResourceID)
		assert.JSONEq(t,
			`{"filename":"redirects.tsv","overwrite":true,"totalLines":50,"imported":50,"skipped":0,"errorCount":0}`,
			string(event.Data),
		)
	})

	t.Run("the error sample is capped while the count stays exact", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		// Twice the sample cap, all invalid
		invalidRows := model.ActivityImportErrorSampleMax * 2
		rows := make([]string, 0, invalidRows)
		for i := 0; i < invalidRows; i++ {
			rows = append(rows, fmt.Sprintf("NOT_A_TYPE\t/old%d\t/new%d\t301", i, i))
		}

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(rows...), ImportRedirectOptions{})
		assert.NoError(t, err)
		assert.Equal(t, invalidRows, result.ErrorCount)

		var payload model.ActivityImport
		assert.NoError(t, json.Unmarshal(lastActivityEvent(t, db).Data, &payload))

		assert.Equal(t, invalidRows, payload.ErrorCount)
		assert.Len(t, payload.ErrorSample, model.ActivityImportErrorSampleMax)
		// A truncated sample stays detectable
		assert.Greater(t, payload.ErrorCount, len(payload.ErrorSample))
		assert.Equal(t, string(ImportErrorInvalidType), payload.ErrorSample[0].Reason)
	})

	t.Run("an empty file still records the import", func(t *testing.T) {
		ctrl, _, db, svc := setupRedirectImportServiceTest(t)
		defer ctrl.Finish()

		result, err := svc.Import(context.Background(), "ns", "proj", tsvFile(), ImportRedirectOptions{})
		assert.NoError(t, err)
		assert.Zero(t, result.TotalLines)

		event := lastActivityEvent(t, db)
		assert.NotNil(t, event)
		assert.Equal(t, model.ActivityActionImport, event.Action)
	})
}
