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
	"gorm.io/gorm/clause"
)

const (
	// MaxImportFileSize is the largest upload accepted by the redirect import.
	MaxImportFileSize = 200 * 1024 * 1024

	// importChunkSize is how many parsed rows are resolved against the database at once.
	// The import streams the file chunk by chunk so memory stays flat regardless of file size.
	importChunkSize = 5000

	// importInsertBatchSize is the number of rows per multi-row INSERT.
	importInsertBatchSize = 1000

	// maxReportedErrors caps the errors carried back to the caller. ErrorCount stays exact,
	// so a truncated list is detectable with errorCount > len(errors).
	maxReportedErrors = 1000
)

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
	Import(ctx context.Context, namespaceCode, projectCode string, reader io.Reader, opts ImportRedirectOptions) (*ImportRedirectResult, error)
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
		return fmt.Errorf("file too large: maximum size is %dMB, got %.2fMB",
			MaxImportFileSize/(1024*1024), float64(size)/(1024*1024))
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

// rowParser streams a CSV/TSV import file one row at a time. It owns the
// cross-file state (line counter and the sources already seen) so that callers
// never need to hold the whole file in memory.
type rowParser struct {
	csv         *csv.Reader
	seenSources map[string]int // source -> first line number
	lineNum     int
}

// newRowParser validates the header and returns a parser positioned on the first data row.
func newRowParser(reader io.Reader) (*rowParser, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = '\t'
	csvReader.LazyQuotes = true
	csvReader.FieldsPerRecord = -1 // Allow variable number of fields per row

	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	expectedColumns := []string{"type", "source", "target", "status"}
	if len(header) != len(expectedColumns) {
		return nil, fmt.Errorf("invalid header: expected %d columns (type, source, target, status), got %d", len(expectedColumns), len(header))
	}
	for i, col := range expectedColumns {
		if strings.ToLower(strings.TrimSpace(header[i])) != col {
			return nil, fmt.Errorf("invalid header: column %d should be '%s', got '%s'", i+1, col, header[i])
		}
	}

	return &rowParser{csv: csvReader, seenSources: make(map[string]int), lineNum: 1}, nil
}

// next returns the next data row. It reports io.EOF once the file is exhausted.
// A non-nil ImportRedirectError means the line was rejected and parsing can continue.
func (p *rowParser) next() (*ParsedRedirectRow, *ImportRedirectError, error) {
	record, err := p.csv.Read()
	if err == io.EOF {
		return nil, nil, io.EOF
	}
	p.lineNum++

	if err != nil {
		return nil, &ImportRedirectError{
			Line:    p.lineNum,
			Reason:  ImportErrorInvalidFormat,
			Message: fmt.Sprintf("failed to read line: %v", err),
		}, nil
	}

	if len(record) != 4 {
		return nil, &ImportRedirectError{
			Line:    p.lineNum,
			Reason:  ImportErrorInvalidFormat,
			Message: fmt.Sprintf("expected 4 columns, got %d", len(record)),
		}, nil
	}

	redirectType, errType := parseRedirectType(strings.TrimSpace(record[0]))
	if errType != nil {
		return nil, &ImportRedirectError{
			Line:    p.lineNum,
			Reason:  ImportErrorInvalidType,
			Message: errType.Error(),
		}, nil
	}

	source := strings.TrimSpace(record[1])
	target := strings.TrimSpace(record[2])

	if source == "" {
		return nil, &ImportRedirectError{
			Line:    p.lineNum,
			Target:  target,
			Reason:  ImportErrorEmptySource,
			Message: "source cannot be empty",
		}, nil
	}
	if target == "" {
		return nil, &ImportRedirectError{
			Line:    p.lineNum,
			Source:  source,
			Reason:  ImportErrorEmptyTarget,
			Message: "target cannot be empty",
		}, nil
	}

	redirectStatus, errStatus := parseRedirectStatus(strings.TrimSpace(record[3]))
	if errStatus != nil {
		return nil, &ImportRedirectError{
			Line:    p.lineNum,
			Source:  source,
			Target:  target,
			Reason:  ImportErrorInvalidStatus,
			Message: errStatus.Error(),
		}, nil
	}

	if firstLine, exists := p.seenSources[source]; exists {
		return nil, &ImportRedirectError{
			Line:    p.lineNum,
			Source:  source,
			Target:  target,
			Reason:  ImportErrorDuplicateInFile,
			Message: fmt.Sprintf("duplicate source in file, first occurrence at line %d", firstLine),
		}, nil
	}
	p.seenSources[source] = p.lineNum

	return &ParsedRedirectRow{
		LineNum: p.lineNum,
		Type:    redirectType,
		Source:  source,
		Target:  target,
		Status:  redirectStatus,
	}, nil, nil
}

// ParseFile parses the whole CSV/TSV file and returns validated rows and parse errors.
// Import streams instead of using this; it is kept for callers that only want to
// validate a file without touching the database.
func (s *redirectImportService) ParseFile(reader io.Reader) ([]ParsedRedirectRow, []ImportRedirectError, error) {
	parser, err := newRowParser(reader)
	if err != nil {
		return nil, nil, err
	}

	var rows []ParsedRedirectRow
	var errors []ImportRedirectError
	for {
		row, rowErr, readErr := parser.next()
		if readErr == io.EOF {
			break
		}
		if rowErr != nil {
			errors = append(errors, *rowErr)
			continue
		}
		rows = append(rows, *row)
	}

	return rows, errors, nil
}

// Import streams the file and imports it as redirect drafts. Rows are resolved and
// written in chunks, so both memory and the number of database round-trips stay
// proportional to the chunk size rather than to the file size.
func (s *redirectImportService) Import(ctx context.Context, namespaceCode, projectCode string, reader io.Reader, opts ImportRedirectOptions) (*ImportRedirectResult, error) {
	s.ctx.Logger.Info("redirect import started", "namespace", namespaceCode, "project", projectCode, "overwrite", opts.Overwrite)

	parser, err := newRowParser(reader)
	if err != nil {
		return nil, err
	}

	result := &ImportRedirectResult{
		Success: true,
		Errors:  make([]ImportRedirectError, 0),
	}

	err = s.redirectDraftRepo.GetTx(ctx).Transaction(func(tx *gorm.DB) error {
		chunk := make([]ParsedRedirectRow, 0, importChunkSize)

		for {
			row, rowErr, readErr := parser.next()
			if readErr == io.EOF {
				break
			}
			result.TotalLines++

			if rowErr != nil {
				addImportError(result, *rowErr)
				continue
			}

			chunk = append(chunk, *row)
			if len(chunk) < importChunkSize {
				continue
			}
			if err := s.importChunk(ctx, tx, namespaceCode, projectCode, chunk, opts, result); err != nil {
				return err
			}
			chunk = chunk[:0]
		}

		if len(chunk) > 0 {
			return s.importChunk(ctx, tx, namespaceCode, projectCode, chunk, opts, result)
		}
		return nil
	})
	if err != nil {
		s.ctx.Logger.Error("redirect import failed", "namespace", namespaceCode, "project", projectCode, "error", err)
		return nil, err
	}

	result.Success = result.ErrorCount == 0
	s.ctx.Logger.Info("redirect import completed", "namespace", namespaceCode, "project", projectCode,
		"lines", result.TotalLines, "imported", result.ImportedCount, "skipped", result.SkippedCount, "errors", result.ErrorCount)
	return result, nil
}

// importChunk resolves a batch of rows against the database and writes them.
// It issues a fixed number of queries per chunk instead of one per row.
func (s *redirectImportService) importChunk(
	ctx context.Context,
	tx *gorm.DB,
	namespaceCode, projectCode string,
	rows []ParsedRedirectRow,
	opts ImportRedirectOptions,
	result *ImportRedirectResult,
) error {
	// Validate each row before hitting the database
	valid := make([]ParsedRedirectRow, 0, len(rows))
	sources := make([]string, 0, len(rows))
	for _, row := range rows {
		if err := s.ctx.Validator.Struct(rowToRedirect(row)); err != nil {
			addImportError(result, ImportRedirectError{
				Line:    row.LineNum,
				Source:  row.Source,
				Target:  row.Target,
				Reason:  ImportErrorInvalidRedirect,
				Message: fmt.Sprintf("invalid data: %v", err),
			})
			continue
		}
		valid = append(valid, row)
		sources = append(sources, row.Source)
	}
	if len(valid) == 0 {
		return nil
	}

	// Resolve what already exists for these sources: two queries for the whole chunk
	redirectBySource, draftBySource, err := s.lookupExisting(ctx, tx, namespaceCode, projectCode, sources)
	if err != nil {
		return err
	}

	var (
		newRedirects []model.Redirect      // brand new redirects, drafts created once their IDs are known
		newRows      []ParsedRedirectRow   // parallel to newRedirects
		newDrafts    []model.RedirectDraft // drafts attached to an already published redirect
		draftUpdates []model.RedirectDraft // existing drafts whose content changes
	)

	for _, row := range valid {
		newRedirect := rowToRedirect(row)
		existing, hasRedirect := redirectBySource[row.Source]
		draft, hasDraft := draftBySource[row.Source]

		if (hasRedirect || hasDraft) && !opts.Overwrite {
			addImportError(result, ImportRedirectError{
				Line:    row.LineNum,
				Source:  row.Source,
				Target:  row.Target,
				Reason:  ImportErrorSourceAlreadyExists,
				Message: "source already exists and overwrite is disabled",
			})
			continue
		}

		switch {
		case hasRedirect && existing.RedirectDraft != nil:
			if redirectsAreEqual(existing.RedirectDraft.NewRedirect, newRedirect) {
				result.SkippedCount++
				continue
			}
			updated := *existing.RedirectDraft
			updated.NewRedirect = newRedirect
			draftUpdates = append(draftUpdates, updated)

		case hasRedirect:
			if redirectsAreEqual(existing.Redirect, newRedirect) {
				result.SkippedCount++
				continue
			}
			newDrafts = append(newDrafts, model.RedirectDraft{
				NamespaceCode: namespaceCode,
				ProjectCode:   projectCode,
				OldRedirectID: types.Ptr(existing.ID),
				ChangeType:    model.DraftChangeTypeUpdate,
				NewRedirect:   newRedirect,
			})

		case hasDraft:
			if redirectsAreEqual(draft.NewRedirect, newRedirect) {
				result.SkippedCount++
				continue
			}
			updated := *draft
			updated.NewRedirect = newRedirect
			draftUpdates = append(draftUpdates, updated)

		default:
			newRedirects = append(newRedirects, model.Redirect{
				NamespaceCode: namespaceCode,
				ProjectCode:   projectCode,
				IsPublished:   types.Ptr(false),
			})
			newRows = append(newRows, row)
		}
	}

	imported := len(newRedirects) + len(newDrafts) + len(draftUpdates)

	// Create the new redirects first, then the drafts pointing at their generated IDs
	if len(newRedirects) > 0 {
		if err := tx.CreateInBatches(&newRedirects, importInsertBatchSize).Error; err != nil {
			return fmt.Errorf("failed to create redirects: %w", err)
		}
		for i, row := range newRows {
			newDrafts = append(newDrafts, model.RedirectDraft{
				NamespaceCode: namespaceCode,
				ProjectCode:   projectCode,
				OldRedirectID: types.Ptr(newRedirects[i].ID),
				ChangeType:    model.DraftChangeTypeCreate,
				NewRedirect:   rowToRedirect(row),
			})
		}
	}

	if len(newDrafts) > 0 {
		if err := tx.CreateInBatches(&newDrafts, importInsertBatchSize).Error; err != nil {
			return fmt.Errorf("failed to create redirect drafts: %w", err)
		}
	}

	// Existing drafts are updated with a batched upsert on their primary key
	if len(draftUpdates) > 0 {
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"new_type", "new_source", "new_target", "new_status", "updated_at"}),
		}).CreateInBatches(&draftUpdates, importInsertBatchSize).Error; err != nil {
			return fmt.Errorf("failed to update redirect drafts: %w", err)
		}
	}

	result.ImportedCount += imported
	return nil
}

// lookupExisting fetches, in two queries, the published redirects and the standalone
// drafts that already use any of the given sources.
func (s *redirectImportService) lookupExisting(
	ctx context.Context,
	tx *gorm.DB,
	namespaceCode, projectCode string,
	sources []string,
) (map[string]*model.Redirect, map[string]*model.RedirectDraft, error) {
	var redirects []model.Redirect
	if err := tx.WithContext(ctx).
		Preload("RedirectDraft").
		Where("namespace_code = ? AND project_code = ? AND source IN ?", namespaceCode, projectCode, sources).
		Find(&redirects).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to load existing redirects: %w", err)
	}
	redirectBySource := make(map[string]*model.Redirect, len(redirects))
	for i := range redirects {
		if redirects[i].Redirect == nil {
			continue
		}
		redirectBySource[redirects[i].Source] = &redirects[i]
	}

	var drafts []model.RedirectDraft
	if err := tx.WithContext(ctx).
		Where("namespace_code = ? AND project_code = ? AND new_source IN ? AND change_type != ?",
			namespaceCode, projectCode, sources, model.DraftChangeTypeDelete).
		Find(&drafts).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to load existing redirect drafts: %w", err)
	}
	draftBySource := make(map[string]*model.RedirectDraft, len(drafts))
	for i := range drafts {
		if drafts[i].NewRedirect == nil {
			continue
		}
		draftBySource[drafts[i].NewRedirect.Source] = &drafts[i]
	}

	return redirectBySource, draftBySource, nil
}

func rowToRedirect(row ParsedRedirectRow) *commonTypes.Redirect {
	return &commonTypes.Redirect{
		Type:   row.Type,
		Source: row.Source,
		Target: row.Target,
		Status: row.Status,
	}
}

// addImportError records an error, keeping the reported list bounded while
// ErrorCount keeps counting every occurrence.
func addImportError(result *ImportRedirectResult, e ImportRedirectError) {
	result.ErrorCount++
	if len(result.Errors) < maxReportedErrors {
		result.Errors = append(result.Errors, e)
	}
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
