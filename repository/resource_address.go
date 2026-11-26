package repository

import (
	"fmt"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

// findAddressesByIDs reads addresses by primary key in chunks, to stay under the
// placeholder limit. sourceColumn is set by the caller, never from user input.
// withOptions also reads the options column, which pages do not have.
func findAddressesByIDs(query *gorm.DB, sourceColumn string, withOptions bool, ids []int64, batchSize int) ([]model.ResourceAddress, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if batchSize <= 0 {
		batchSize = len(ids)
	}

	columns := fmt.Sprintf("id, type, %s AS source", sourceColumn)
	if withOptions {
		columns += ", options"
	}

	addresses := make([]model.ResourceAddress, 0, len(ids))

	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		var chunk []model.ResourceAddress
		err := query.Session(&gorm.Session{}).
			Select(columns).
			Where("id IN ?", ids[start:end]).
			Scan(&chunk).Error
		if err != nil {
			return nil, err
		}

		addresses = append(addresses, chunk...)
	}

	return addresses, nil
}
