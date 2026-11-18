package domain

import (
	"encoding/json"
	"fmt"
)

// FileBytes is the project as a save writes it. Stamping on every save is also the migration: a
// pre-versioning file gains the schema field the first time it is written back.
func FileBytes(p Project) ([]byte, error) {
	p.Schema = RequiredSchema(p)
	return json.MarshalIndent(p, "", "  ")
}

// ParseImport reads a project file handed to an import, refusing one a newer ordo wrote.
func ParseImport(raw []byte) (Project, error) {
	var p Project
	if err := json.Unmarshal(raw, &p); err != nil {
		return Project{}, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := CheckSchema(p); err != nil {
		return Project{}, err
	}
	// The repair can move a category to the top level, so siblings are only known after it.
	NormalizeNesting(&p)
	for i := range p.Categories {
		p.Categories[i].Name = UniqueCategoryName(p.Categories[:i], p.Categories[i].ParentID, p.Categories[i].Name)
	}
	return p, nil
}
