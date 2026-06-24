package schema

import (
	"encoding/json"
	_ "embed"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed cognitive.schema.json
var schemaData string

var compiled *jsonschema.Schema

func init() {
	var doc interface{}
	if err := json.Unmarshal([]byte(schemaData), &doc); err != nil {
		return
	}

	compiler := jsonschema.NewCompiler()
	compiler.AddResource("https://cognitive-os.org/schemas/cognitive.schema.json", doc)

	sch, err := compiler.Compile("https://cognitive-os.org/schemas/cognitive.schema.json")
	if err != nil {
		return
	}
	compiled = sch
}

func Validate(doc map[string]interface{}) error {
	if compiled == nil {
		return nil
	}
	if err := compiled.Validate(doc); err != nil {
		return fmt.Errorf("schema validation failed: %v", err)
	}
	return nil
}
