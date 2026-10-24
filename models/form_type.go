package models

import (
	"encoding/json"
	"fmt"
)

type FormType struct {
	Name           string         `json:"name"`
	Audience       Audience       `json:"audience"`
	CollectionMode CollectionMode `json:"collection_mode"`
}

func ParseFormTypes(value any) ([]FormType, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("form_types: %w", err)
	}
	var types []FormType
	if err := json.Unmarshal(raw, &types); err != nil {
		return nil, fmt.Errorf("form_types: %w", err)
	}
	seen := map[string]bool{}
	for _, t := range types {
		if t.Name == "" {
			return nil, fmt.Errorf("form_types: a type has no name")
		}
		if seen[t.Name] {
			return nil, fmt.Errorf("form_types: %q is named twice", t.Name)
		}
		seen[t.Name] = true
		if t.Audience != AudienceMember && t.Audience != AudiencePublic {
			return nil, fmt.Errorf("form_types: %q: audience %q is not member or public", t.Name, t.Audience)
		}
		if t.CollectionMode != CollectionSelf && t.CollectionMode != CollectionInterviewer {
			return nil, fmt.Errorf("form_types: %q: collection_mode %q is not self or interviewer", t.Name, t.CollectionMode)
		}
		if err := ValidateAudienceCollection(t.Audience, t.CollectionMode); err != nil {
			return nil, fmt.Errorf("form_types: %q: %w", t.Name, err)
		}
	}
	if types == nil {
		types = []FormType{}
	}
	return types, nil
}
