package repository

import (
	"fmt"

	"github.com/flectolab/flecto-manager/model"
	"gorm.io/gorm"
)

// findAddressesByIDs reads addresses by primary key in chunks, to stay under the
// placeholder limit. sourceColumn is set by the caller, never from user input.
func findAddressesByIDs(query *gorm.DB, sourceColumn string, ids []int64, batchSize int) ([]model.ResourceAddress, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if batchSize <= 0 {
		batchSize = len(ids)
	}

	addresses := make([]model.ResourceAddress, 0, len(ids))

	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		var chunk []model.ResourceAddress
		err := query.Session(&gorm.Session{}).
			Select(fmt.Sprintf("id, type, %s AS source", sourceColumn)).
			Where("id IN ?", ids[start:end]).
			Scan(&chunk).Error
		if err != nil {
			return nil, err
		}

		addresses = append(addresses, chunk...)
	}

	return addresses, nil
}
