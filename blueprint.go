package mwanachamaforms

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/aosanya/mwanachama-backend-shared/spec"
)

//go:embed forms.blueprint.json
var blueprintJSON []byte

//go:embed forms.forms.json
var domainJSON []byte

var loadBlueprint = sync.OnceValues(func() (*spec.Blueprint, error) {
	return spec.ParseBlueprint(blueprintJSON)
})

func Blueprint() (*spec.Blueprint, error) { return loadBlueprint() }

func LoadSpec(path string) (*spec.Spec, error) {
	b, err := Blueprint()
	if err != nil {
		return nil, err
	}
	return b.Load(path)
}

func ParseSpec(raw []byte) (*spec.Spec, error) {
	b, err := Blueprint()
	if err != nil {
		return nil, err
	}
	return b.Parse(raw)
}

// SpecFor is how a mount chooses its instance: the shipped domain spec names
// one as an example, and this replaces it, so two mounts of this module in
// one database never reach for the same physical table.
func SpecFor(instance string) (*spec.Spec, error) {
	var doc map[string]any
	if err := json.Unmarshal(domainJSON, &doc); err != nil {
		return nil, fmt.Errorf("forms spec: %w", err)
	}
	doc["instance"] = instance

	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("forms spec: %w", err)
	}
	return ParseSpec(raw)
}
