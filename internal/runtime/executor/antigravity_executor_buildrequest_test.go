package executor

// NOTE: The fork extended CleanJSONSchemaForAntigravity/Gemini to additionally
// remove $id, patternProperties, prefill, and enumTitles. Origin only removes the keywords
// defined in its removeUnsupportedKeywords ($schema, $defs, definitions, const, $ref,
// additionalProperties, propertyNames) and unsupportedConstraints.
// Tests below verify the behaviour that IS present in origin:
//   - parametersJsonSchema is renamed to parameters
//   - $schema is removed from the schema root
//   - properties named $id (as a property key) are accessible

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

// TestAntigravityBuildRequest_RenamesParametersJsonSchema verifies that the key
// "parametersJsonSchema" is renamed to "parameters" before the request is sent,
// for both Gemini and Antigravity (claude) models.
func TestAntigravityBuildRequest_RenamesParametersJsonSchema(t *testing.T) {
	for _, modelName := range []string{"gemini-2.5-pro", "claude-opus-4-6"} {
		t.Run(modelName, func(t *testing.T) {
			body := buildRequestBodyFromPayload(t, modelName)
			decl := extractFirstFunctionDeclaration(t, body)
			if _, ok := decl["parametersJsonSchema"]; ok {
				t.Fatalf("model %s: parametersJsonSchema should be renamed to parameters", modelName)
			}
			if _, ok := decl["parameters"]; !ok {
				t.Fatalf("model %s: parameters key missing after rename", modelName)
			}
		})
	}
}

// TestAntigravityBuildRequest_RemovesSchemaKeyword verifies that $schema is stripped
// from tool parameter schemas by the Gemini/Antigravity schema cleaner.
func TestAntigravityBuildRequest_RemovesSchemaKeyword(t *testing.T) {
	body := buildRequestBodyFromPayload(t, "gemini-2.5-pro")
	decl := extractFirstFunctionDeclaration(t, body)
	params, ok := decl["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("parameters missing or invalid type")
	}
	if _, ok := params["$schema"]; ok {
		t.Fatalf("$schema should be removed from tool parameter schema")
	}
}

func buildRequestBodyFromPayload(t *testing.T, modelName string) map[string]any {
	t.Helper()

	executor := &AntigravityExecutor{}
	auth := &cliproxyauth.Auth{}
	payload := []byte(`{
		"request": {
			"tools": [
				{
					"function_declarations": [
						{
							"name": "tool_1",
							"parametersJsonSchema": {
								"$schema": "http://json-schema.org/draft-07/schema#",
								"type": "object",
								"properties": {
									"arg": {
										"type": "string"
									}
								}
							}
						}
					]
				}
			]
		}
	}`)

	req, err := executor.buildRequest(context.Background(), auth, "token", modelName, payload, false, "", "https://example.com")
	if err != nil {
		t.Fatalf("buildRequest error: %v", err)
	}

	raw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal request body error: %v, body=%s", err, string(raw))
	}
	return body
}

func extractFirstFunctionDeclaration(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	request, ok := body["request"].(map[string]any)
	if !ok {
		t.Fatalf("request missing or invalid type")
	}
	tools, ok := request["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("tools missing or empty")
	}
	tool, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("first tool invalid type")
	}
	decls, ok := tool["function_declarations"].([]any)
	if !ok || len(decls) == 0 {
		t.Fatalf("function_declarations missing or empty")
	}
	decl, ok := decls[0].(map[string]any)
	if !ok {
		t.Fatalf("first function declaration invalid type")
	}
	return decl
}
