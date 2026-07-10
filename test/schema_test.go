package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/AxeForging/structlint/internal/config"
)

func loadSchema(t *testing.T) map[string]any {
	t.Helper()
	path := filepath.Join(repoRoot(t), "schema", "structlint.schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	return m
}

func TestSchema_IsValidJSON(t *testing.T) {
	s := loadSchema(t)

	if got, _ := s["$schema"].(string); !strings.Contains(got, "draft-07") {
		t.Errorf("$schema should reference draft-07, got %q", got)
	}
	if got, _ := s["additionalProperties"].(bool); got != false {
		t.Error("top-level additionalProperties must be false to mirror strict parse")
	}
	if _, ok := s["properties"].(map[string]any); !ok {
		t.Error("schema must declare properties")
	}
}

// TestSchema_RejectsUnknownProperties walks the schema recursively and asserts
// every `object` node has additionalProperties: false. If a new object node
// is added without that guard, editors would silently accept typos the
// strict parser rejects.
func TestSchema_RejectsUnknownProperties(t *testing.T) {
	s := loadSchema(t)
	walkObjects(t, s, "$")
}

func walkObjects(t *testing.T, node any, path string) {
	m, ok := node.(map[string]any)
	if !ok {
		return
	}
	if typ, _ := m["type"].(string); typ == "object" {
		ap, has := m["additionalProperties"]
		if !has {
			t.Errorf("%s: object node missing additionalProperties: false", path)
		} else if b, ok := ap.(bool); !ok || b != false {
			t.Errorf("%s: object node has additionalProperties=%v, want false", path, ap)
		}
	}
	// Recurse through properties.
	if props, ok := m["properties"].(map[string]any); ok {
		for k, v := range props {
			walkObjects(t, v, path+".properties."+k)
		}
	}
	// Recurse into items (array element schemas).
	if items, ok := m["items"].(map[string]any); ok {
		walkObjects(t, items, path+".items")
	}
	// Recurse into definitions.
	if defs, ok := m["definitions"].(map[string]any); ok {
		for k, v := range defs {
			walkObjects(t, v, path+".definitions."+k)
		}
	}
}

// TestSchema_CoversAllConfigFields reflects over config.Config and asserts
// every yaml tag exists at the corresponding schema path. A new config
// field without a schema entry fails here with an actionable message.
func TestSchema_CoversAllConfigFields(t *testing.T) {
	s := loadSchema(t)
	rootProps, _ := s["properties"].(map[string]any)
	if rootProps == nil {
		t.Fatal("schema has no top-level properties")
	}
	checkStructFields(t, reflect.TypeOf(config.Config{}), rootProps, "$.properties")
}

// checkStructFields walks a struct type's yaml tags and asserts they appear
// under the given schema properties map. For nested structs and rule-item
// structs, it recurses into their yaml tags too.
func checkStructFields(t *testing.T, typ reflect.Type, props map[string]any, path string) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := strings.Split(field.Tag.Get("yaml"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		schemaEntry, ok := props[tag].(map[string]any)
		if !ok {
			t.Errorf("%s: yaml tag %q from %s.%s missing in schema", path, tag, typ.Name(), field.Name)
			continue
		}
		// Recurse into nested structs and slices of structs.
		ft := field.Type
		switch ft.Kind() {
		case reflect.Struct:
			if inner, ok := schemaEntry["properties"].(map[string]any); ok {
				checkStructFields(t, ft, inner, path+"."+tag+".properties")
			}
		case reflect.Slice:
			if ft.Elem().Kind() == reflect.Struct {
				if items, ok := schemaEntry["items"].(map[string]any); ok {
					if inner, ok := items["properties"].(map[string]any); ok {
						checkStructFields(t, ft.Elem(), inner, path+"."+tag+".items.properties")
					}
				}
			}
		}
	}
}

// TestSchema_SelfConfigValidates is the binary-based test — the built binary
// running against this repo must succeed with the new schema/ directory in
// place, proving the .structlint.yaml update landed with the schema file.
func TestSchema_SelfConfigValidates(t *testing.T) {
	bin := buildBinary(t)
	out, err := runBinaryInDir(t, bin, repoRoot(t), "validate", "--silent")
	if err != nil {
		t.Fatalf("self-validate failed with schema/ present: err=%v out:\n%s", err, out)
	}
}
