package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	commonTypes "github.com/flectolab/flecto-manager/common/types"
	appContext "github.com/flectolab/flecto-manager/context"
	"github.com/flectolab/flecto-manager/model"
	"github.com/flectolab/flecto-manager/repository"
	"github.com/flectolab/flecto-manager/types"
	"gorm.io/gorm"
)

const MaxImportFileSize = 2 * 1024 * 1024

// ImportErrorReason represents the reason why a redirect import failed
type ImportErrorReason string

const (
	ImportErrorInvalidFormat       ImportErrorReason = "INVALID_FORMAT"
	ImportErrorInvalidRedirect     ImportErrorReason = "INVALID_REDIRECT"
	ImportErrorInvalidType         ImportErrorReason = "INVALID_TYPE"
	ImportErrorInvalidStatus       ImportErrorReason = "INVALID_STATUS"
	ImportErrorEmptySource         ImportErrorReason = "EMPTY_SOURCE"
	ImportErrorEmptyTarget         ImportErrorReason = "EMPTY_TARGET"
	ImportErrorDuplicateInFile     ImportErrorReason = "DUPLICATE_SOURCE_IN_FILE"
	ImportErrorSourceAlreadyExists ImportErrorReason = "SOURCE_ALREADY_EXISTS"
	ImportErrorDatabaseError       ImportErrorReason = "DATABASE_ERROR"
)

// ImportRedirectError represents a single import error
type ImportRedirectError struct {
	Line    int
	Source  string
	Target  string
	Reason  ImportErrorReason
	Message string
}

// ImportRedirectResult represents the result of an import operation
type ImportRedirectResult struct {
	Success       bool
	TotalLines    int
	ImportedCount int
	SkippedCount  int
	ErrorCount    int
	Errors        []ImportRedirectError
}

// ImportRedirectOptions contains options for the import operation
type ImportRedirectOptions struct {
	Overwrite bool
}

// ParsedRedirectRow represents a parsed row from the import file
type ParsedRedirectRow struct {
	LineNum int
	Type    commonTypes.RedirectType
	Source  string
	Target  string
	Status  commonTypes.RedirectStatus
}

// RedirectImportService handles redirect import operations
type RedirectImportService interface {
	GetTx(ctx context.Context) *gorm.DB
	GetQuery(ctx context.Context) *gorm.DB
	ValidateFile(filename string, contentType string, size int64) error
	ParseFile(reader io.Reader) ([]ParsedRedirectRow, []ImportRedirectError, error)
	Import(ctx context.Context, namespaceCode, projectCode string, rows []ParsedRedirectRow, opts ImportRedirectOptions) (*ImportRedirectResult, error)
}

type redirectImportService struct {
	ctx               *appContext.Context
	redirectDraftRepo repository.RedirectDraftRepository
}

// NewRedirectImportService creates a new RedirectImportService
func NewRedirectImportService(ctx *appContext.Context, redirectDraftRepo repository.RedirectDraftRepository) RedirectImportService {
	return &redirectImportService{
		ctx:               ctx,
		redirectDraftRepo: redirectDraftRepo,
	}
}

func (s *redirectImportService) GetTx(ctx context.Context) *gorm.DB {
	return s.redirectDraftRepo.GetTx(ctx)
}

func (s *redirectImportService) GetQuery(ctx context.Context) *gorm.DB {
	return s.redirectDraftRepo.GetQuery(ctx)
}

// ValidateFile validates the file metadata before parsing
func (s *redirectImportService) ValidateFile(filename string, contentType string, size int64) error {
	// Validate file size
	if size > MaxImportFileSize {
		return fmt.Errorf("file too large: maximum size is 2MB, got %.2fMB", float64(size)/(1024*1024))
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".csv" && ext != ".tsv" {
		return fmt.Errorf("invalid file type: only .csv and .tsv files are allowed")
	}

	// Validate content type
	ct := strings.ToLower(contentType)
	allowedContentTypes := []string{
		"text/csv",
		"text/tab-separated-values",
		"text/plain",
		"application/csv",
		"application/octet-stream",
	}
	for _, allowed := range allowedContentTypes {
		if strings.HasPrefix(ct, allowed) {
			return nil
		}
	}
	return fmt.Errorf("invalid content type: %s", contentType)
}

// ParseFile parses the CSV/TSV file and returns validated rows and parse errors
func (s *redirectImportService) ParseFile(reader io.Reader) ([]ParsedRedirectRow, []ImportRedirectError, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = '\t'
	csvReader.LazyQuotes = true
	csvReader.FieldsPerRecord = -1 // Allow variable number of fields per row

	// Read and validate header
	header, err := csvReader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read header: %w", err)
	}

	expectedColumns := []string{"type", "source", "target", "status"}
	if len(header) != len(expectedColumns) {
		return nil, nil, fmt.Errorf("invalid header: expected %d columns (type, source, target, status), got %d", len(expectedColumns), len(header))
	}
	for i, col := range expectedColumns {
		if strings.ToLower(strings.TrimSpace(header[i])) != col {
			return nil, nil, fmt.Errorf("invalid header: column %d should be '%s', got '%s'", i+1, col, header[i])
		}
	}

	var rows []ParsedRedirectRow
	var errors []ImportRedirectError
	seenSources := make(map[string]int) // source -> first line number

	lineNum := 1
	for {
		record, errRead := csvReader.Read()
		if errRead == io.EOF {
			break
		}
		lineNum++

		if errRead != nil {
			errors = append(errors, ImportRedirectError{
				Line:    lineNum,
				Reason:  ImportErrorInvalidFormat,
				Message: fmt.Sprintf("failed to read line: %v", errRead),
			})
			continue
		}

		if len(record) != 4 {
			errors = append(errors, ImportRedirectError{
				Line:    lineNum,
				Reason:  ImportErrorInvalidFormat,
				Message: fmt.Sprintf("expected 4 columns, got %d", len(record)),
			})
			continue
		}

		// Parse type
		redirectType, errType := parseRedirectType(strings.TrimSpace(record[0]))
		if errType != nil {
			errors = append(errors, ImportRedirectError{
				Line:    lineNum,
				Reason:  ImportErrorInvalidType,
				Message: errType.Error(),
			})
			continue
		}

		source := strings.TrimSpace(record[1])
		target := strings.TrimSpace(record[2])

		if source == "" {
			errors = append(errors, ImportRedirectError{
				Line:    lineNum,
				Target:  target,
				Reason:  ImportErrorEmptySource,
				Message: "source cannot be empty",
			})
			continue
		}
		if target == "" {
			errors = append(errors, ImportRedirectError{
				Line:    lineNum,
				Source:  source,
				Reason:  ImportErrorEmptyTarget,
				Message: "target cannot be empty",
			})
			continue
		}

		// Parse status
		redirectStatus, errStatus := parseRedirectStatus(strings.TrimSpace(record[3]))
		if errStatus != nil {
			errors = append(errors, ImportRedirectError{
				Line:    lineNum,
				Source:  source,
				Target:  target,
				Reason:  ImportErrorInvalidStatus,
				Message: errStatus.Error(),
			})
			continue
		}

		// Check for duplicate sources within the file
		if firstLine, exists := seenSources[source]; exists {
			errors = append(errors, ImportRedirectError{
				Line:    lineNum,
				Source:  source,
				Target:  target,
				Reason:  ImportErrorDuplicateInFile,
				Message: fmt.Sprintf("duplicate source in file, first occurrence at line %d", firstLine),
			})
			continue
		}
		seenSources[source] = lineNum

		rows = append(rows, ParsedRedirectRow{
			LineNum: lineNum,
			Type:    redirectType,
			Source:  source,
			Target:  target,
			Status:  redirectStatus,
		})
	}

	return rows, errors, nil
}

// Import imports the parsed rows into the database
func (s *redirectImportService) Import(ctx context.Context, namespaceCode, projectCode string, rows []ParsedRedirectRow, opts ImportRedirectOptions) (*ImportRedirectResult, error) {
	result := &ImportRedirectResult{
		Success:    true,
		TotalLines: len(rows),
		Errors:     make([]ImportRedirectError, 0),
	}

	if len(rows) == 0 {
		return result, nil
	}

	// Collect all sources for batch availability check
	sources := make([]string, len(rows))
	for i, row := range rows {
		sources[i] = row.Source
	}

	// Check source availability for all sources
	unavailableSources, err := s.checkSourcesAvailability(ctx, namespaceCode, projectCode, sources)
	if err != nil {
		return nil, fmt.Errorf("failed to check source availability: %w", err)
	}

	// Filter rows based on availability and overwrite option
	var rowsToImport []ParsedRedirectRow
	for _, row := range rows {
		if _, unavailable := unavailableSources[row.Source]; unavailable {
			if !opts.Overwrite {
				result.Errors = append(result.Errors, ImportRedirectError{
					Line:    row.LineNum,
					Source:  row.Source,
					Target:  row.Target,
					Reason:  ImportErrorSourceAlreadyExists,
					Message: "source already exists and overwrite is disabled",
				})
				result.ErrorCount++
				continue
			}
			// If overwrite is enabled, we'll handle it during import
		}
		rowsToImport = append(rowsToImport, row)
	}

	if len(rowsToImport) == 0 {
		result.Success = result.ErrorCount == 0
		return result, nil
	}

	// Execute import in a single transaction
	err = s.redirectDraftRepo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rowsToImport {
			imported, importErr := s.importRow(ctx, tx, namespaceCode, projectCode, row, unavailableSources)
			if importErr != nil {
				result.Errors = append(result.Errors, *importErr)
				result.ErrorCount++
			} else if imported {
				result.ImportedCount++
			} else {
				result.SkippedCount++
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	result.Success = result.ErrorCount == 0
	return result, nil
}

// checkSourcesAvailability checks which sources already exist
func (s *redirectImportService) checkSourcesAvailability(ctx context.Context, namespaceCode, projectCode string, sources []string) (map[string]bool, error) {
	unavailable := make(map[string]bool)

	for _, source := range sources {
		available, err := s.redirectDraftRepo.CheckSourceAvailability(ctx, namespaceCode, projectCode, source, nil, nil)
		if err != nil {
			return nil, err
		}
		if !available {
			unavailable[source] = true
		}
	}

	return unavailable, nil
}

// importRow imports a single row, returns (imported, error)
func (s *redirectImportService) importRow(ctx context.Context, tx *gorm.DB, namespaceCode, projectCode string, row ParsedRedirectRow, unavailableSources map[string]bool) (bool, *ImportRedirectError) {
	newRedirect := &commonTypes.Redirect{
		Type:   row.Type,
		Source: row.Source,
		Target: row.Target,
		Status: row.Status,
	}
	errValidate := s.ctx.Validator.Struct(newRedirect)
	if errValidate != nil {
		return false, &ImportRedirectError{
			Line:    row.LineNum,
			Source:  row.Source,
			Target:  row.Target,
			Reason:  ImportErrorInvalidRedirect,
			Message: fmt.Sprintf("invalid data: %v", errValidate),
		}
	}

	// Check if source already exists (only reached when overwrite is enabled)
	if _, exists := unavailableSources[row.Source]; exists {
		return s.updateExistingDraft(ctx, tx, namespaceCode, projectCode, row, newRedirect)
	}

	// Create new redirect and draft
	return s.createNewDraft(tx, namespaceCode, projectCode, row, newRedirect)
}

// updateExistingDraft updates an existing draft for a source
func (s *redirectImportService) updateExistingDraft(ctx context.Context, tx *gorm.DB, namespaceCode, projectCode string, row ParsedRedirectRow, newRedirect *commonTypes.Redirect) (bool, *ImportRedirectError) {
	// Find existing redirect with this source
	var existingRedirect model.Redirect
	err := tx.WithContext(ctx).
		Preload("RedirectDraft").
		Where("namespace_code = ? AND project_code = ? AND source = ?", namespaceCode, projectCode, row.Source).
		First(&existingRedirect).Error

	if err == nil && existingRedirect.ID > 0 {
		// Update or create draft for existing published redirect
		if existingRedirect.RedirectDraft != nil {
			// Check if data is identical - skip if no changes
			if redirectsAreEqual(existingRedirect.RedirectDraft.NewRedirect, newRedirect) {
				return false, nil // Skip, no changes
			}
			existingRedirect.RedirectDraft.NewRedirect = newRedirect
			if err = tx.Save(existingRedirect.RedirectDraft).Error; err != nil {
				return false, &ImportRedirectError{
					Line:    row.LineNum,
					Source:  row.Source,
					Target:  row.Target,
					Reason:  ImportErrorDatabaseError,
					Message: fmt.Sprintf("failed to update existing draft: %v", err),
				}
			}
			return true, nil
		}

		// Check if the published redirect already has the same data
		publishedRedirect := &commonTypes.Redirect{
			Type:   existingRedirect.Type,
			Source: existingRedirect.Source,
			Target: existingRedirect.Target,
			Status: existingRedirect.Status,
		}
		if redirectsAreEqual(publishedRedirect, newRedirect) {
			return false, nil // Skip, no changes from published version
		}

		// Create new draft for published redirect
		draft := &model.RedirectDraft{
			NamespaceCode: namespaceCode,
			ProjectCode:   projectCode,
			OldRedirectID: types.Ptr(existingRedirect.ID),
			ChangeType:    model.DraftChangeTypeUpdate,
			NewRedirect:   newRedirect,
		}
		if err = tx.Create(draft).Error; err != nil {
			return false, &ImportRedirectError{
				Line:    row.LineNum,
				Source:  row.Source,
				Target:  row.Target,
				Reason:  ImportErrorDatabaseError,
				Message: fmt.Sprintf("failed to create draft for existing redirect: %v", err),
			}
		}
		return true, nil
	}

	// Check in redirect_drafts for unpublished redirects with matching new_source
	var existingDraft model.RedirectDraft
	err = tx.WithContext(ctx).
		Where("namespace_code = ? AND project_code = ? AND new_source = ? AND change_type != ?",
			namespaceCode, projectCode, row.Source, model.DraftChangeTypeDelete).
		First(&existingDraft).Error

	if err == nil && existingDraft.ID > 0 {
		// Check if data is identical - skip if no changes
		if redirectsAreEqual(existingDraft.NewRedirect, newRedirect) {
			return false, nil // Skip, no changes
		}
		existingDraft.NewRedirect = newRedirect
		if err = tx.Save(&existingDraft).Error; err != nil {
			return false, &ImportRedirectError{
				Line:    row.LineNum,
				Source:  row.Source,
				Target:  row.Target,
				Reason:  ImportErrorDatabaseError,
				Message: fmt.Sprintf("failed to update existing draft: %v", err),
			}
		}
		return true, nil
	}

	// If we get here, the source exists but we couldn't find it (shouldn't happen)
	return s.createNewDraft(tx, namespaceCode, projectCode, row, newRedirect)
}

// redirectsAreEqual compares two redirects to check if they have identical data
func redirectsAreEqual(a, b *commonTypes.Redirect) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Type == b.Type &&
		a.Source == b.Source &&
		a.Target == b.Target &&
		a.Status == b.Status
}

// createNewDraft creates a new redirect and draft
func (s *redirectImportService) createNewDraft(tx *gorm.DB, namespaceCode, projectCode string, row ParsedRedirectRow, newRedirect *commonTypes.Redirect) (bool, *ImportRedirectError) {
	// Create new unpublished redirect
	redirect := &model.Redirect{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		IsPublished:   types.Ptr(false),
	}
	if err := tx.Create(redirect).Error; err != nil {
		return false, &ImportRedirectError{
			Line:    row.LineNum,
			Source:  row.Source,
			Target:  row.Target,
			Reason:  ImportErrorDatabaseError,
			Message: fmt.Sprintf("failed to create redirect: %v", err),
		}
	}

	// Create redirect draft
	draft := &model.RedirectDraft{
		NamespaceCode: namespaceCode,
		ProjectCode:   projectCode,
		OldRedirectID: types.Ptr(redirect.ID),
		ChangeType:    model.DraftChangeTypeCreate,
		NewRedirect:   newRedirect,
	}
	if err := tx.Create(draft).Error; err != nil {
		return false, &ImportRedirectError{
			Line:    row.LineNum,
			Source:  row.Source,
			Target:  row.Target,
			Reason:  ImportErrorDatabaseError,
			Message: fmt.Sprintf("failed to create redirect draft: %v", err),
		}
	}

	return true, nil
}

// Helper functions moved from resolver
func parseRedirectType(s string) (commonTypes.RedirectType, error) {
	switch strings.ToUpper(s) {
	case "BASIC":
		return commonTypes.RedirectTypeBasic, nil
	case "BASIC_HOST":
		return commonTypes.RedirectTypeBasicHost, nil
	case "REGEX":
		return commonTypes.RedirectTypeRegex, nil
	case "REGEX_HOST":
		return commonTypes.RedirectTypeRegexHost, nil
	default:
		return "", fmt.Errorf("invalid redirect type '%s': must be BASIC, BASIC_HOST, REGEX, or REGEX_HOST", s)
	}
}

func parseRedirectStatus(s string) (commonTypes.RedirectStatus, error) {
	switch strings.ToUpper(s) {
	case "MOVED_PERMANENT", "301":
		return commonTypes.RedirectStatusMovedPermanent, nil
	case "FOUND", "302":
		return commonTypes.RedirectStatusFound, nil
	case "TEMPORARY_REDIRECT", "307":
		return commonTypes.RedirectStatusTemporary, nil
	case "PERMANENT_REDIRECT", "308":
		return commonTypes.RedirectStatusPermanent, nil
	default:
		return "", fmt.Errorf("invalid redirect status '%s': must be MOVED_PERMANENT (301), FOUND (302), TEMPORARY_REDIRECT (307), or PERMANENT_REDIRECT (308)", s)
	}
}
