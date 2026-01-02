package database

import (
	"gorm.io/gorm"
)

type SortDirection string

const (
	SortASC  SortDirection = "ASC"
	SortDESC SortDirection = "DESC"
)

type SortInput struct {
	Column    string
	Direction SortDirection
}

// ApplySort applique les tris à une requête GORM
// allowedColumns: map[jsonName]dbColumnName
// tablePrefix: préfixe de table pour les jointures (optionnel, "" si pas de jointure)
func ApplySort(query *gorm.DB, allowedColumns map[string]string, sorts []SortInput, tablePrefix string) *gorm.DB {
	for _, sort := range sorts {
		col, ok := allowedColumns[sort.Column]
		if !ok {
			continue
		}

		dir := SortASC
		if sort.Direction == SortDESC {
			dir = SortDESC
		}

		if tablePrefix != "" {
			col = tablePrefix + "." + col
		}

		query = query.Order(col + " " + string(dir))
	}
	return query
}