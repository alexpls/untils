package docs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

type openAPISpec struct {
	Servers    []openAPIServer        `yaml:"servers"`
	Paths      map[string]openAPIPath `yaml:"paths"`
	Webhooks   map[string]openAPIPath `yaml:"webhooks"`
	Components openAPIComponents      `yaml:"components"`
}

type openAPIServer struct {
	URL string `yaml:"url"`
}

type openAPIPath map[string]openAPIOperation

type openAPIOperation struct {
	OperationID string                        `yaml:"operationId"`
	Summary     string                        `yaml:"summary"`
	Description string                        `yaml:"description"`
	Security    []map[string][]string         `yaml:"security"`
	Parameters  []openAPIParameter            `yaml:"parameters"`
	RequestBody openAPIRequestBodyRef         `yaml:"requestBody"`
	Responses   map[string]openAPIResponseRef `yaml:"responses"`
}

type openAPIParameter struct {
	Name        string           `yaml:"name"`
	In          string           `yaml:"in"`
	Required    bool             `yaml:"required"`
	Description string           `yaml:"description"`
	Schema      openAPISchemaRef `yaml:"schema"`
}

type openAPIComponents struct {
	SecuritySchemes map[string]openAPISecurityScheme `yaml:"securitySchemes"`
	Responses       map[string]openAPIResponseRef    `yaml:"responses"`
	Schemas         map[string]openAPISchemaRef      `yaml:"schemas"`
}

type openAPISecurityScheme struct {
	Description string `yaml:"description"`
}

type openAPIResponseRef struct {
	Ref         string                      `yaml:"$ref"`
	Description string                      `yaml:"description"`
	Content     map[string]openAPIMediaType `yaml:"content"`
}

type openAPIRequestBodyRef struct {
	Ref         string                      `yaml:"$ref"`
	Description string                      `yaml:"description"`
	Required    bool                        `yaml:"required"`
	Content     map[string]openAPIMediaType `yaml:"content"`
}

type openAPIMediaType struct {
	Schema   openAPISchemaRef             `yaml:"schema"`
	Examples map[string]openAPIExampleRef `yaml:"examples"`
}

type openAPIExampleRef struct {
	Summary string    `yaml:"summary"`
	Value   yaml.Node `yaml:"value"`
}

type openAPISchemaRef struct {
	Ref                  string                      `yaml:"$ref"`
	Description          string                      `yaml:"description"`
	Examples             []yaml.Node                 `yaml:"examples"`
	Type                 string                      `yaml:"type"`
	Format               string                      `yaml:"format"`
	Const                any                         `yaml:"const"`
	Enum                 []any                       `yaml:"enum"`
	Nullable             bool                        `yaml:"nullable"`
	Required             []string                    `yaml:"required"`
	Properties           map[string]openAPISchemaRef `yaml:"properties"`
	Items                *openAPISchemaRef           `yaml:"items"`
	AllOf                []openAPISchemaRef          `yaml:"allOf"`
	OneOf                []openAPISchemaRef          `yaml:"oneOf"`
	MaxItems             *int                        `yaml:"maxItems"`
	MinItems             *int                        `yaml:"minItems"`
	AdditionalProperties any                         `yaml:"additionalProperties"`
}

type openAPIMethod struct {
	Name      string
	Path      string
	Method    string
	Operation openAPIOperation
}

type openAPIWebhook struct {
	Name      string
	Key       string
	Method    string
	Operation openAPIOperation
}

func LoadOpenAPIReference(path string) (APIReference, error) {
	return LoadOpenAPIReferenceFS(os.DirFS(filepath.Dir(path)), filepath.Base(path))
}

func LoadOpenAPIReferenceFS(docsFS fs.FS, specPath string) (APIReference, error) {
	return loadOpenAPIReferenceFS(docsFS, specPath, markdownRenderer())
}

func loadOpenAPIReferenceFS(docsFS fs.FS, specPath string, renderer goldmark.Markdown) (APIReference, error) {
	spec, err := loadOpenAPISpec(docsFS, specPath)
	if err != nil {
		return APIReference{}, err
	}
	reference := spec.apiReference()
	if err := renderAPIReferenceMarkdown(&reference, renderer); err != nil {
		return APIReference{}, err
	}
	return reference, nil
}

func loadOpenAPISpec(docsFS fs.FS, specPath string) (openAPISpec, error) {
	content, err := fs.ReadFile(docsFS, specPath)
	if err != nil {
		return openAPISpec{}, err
	}

	var spec openAPISpec
	if err := yaml.Unmarshal(content, &spec); err != nil {
		return openAPISpec{}, err
	}
	return spec, nil
}

func (spec openAPISpec) apiReference() APIReference {
	reference := APIReference{}

	for _, method := range spec.apiMethods() {
		authorization, authorizationDescription := spec.operationAuthorization(method.Operation)
		reference.Methods = append(reference.Methods, APIMethod{
			Name:                     method.Name,
			Anchor:                   operationAnchor(method),
			Description:              method.Operation.Description,
			HTTPMethod:               strings.ToUpper(method.Method),
			Endpoint:                 spec.endpointPath(method.Path),
			Authorization:            authorization,
			AuthorizationDescription: authorizationDescription,
			Parameters:               spec.apiParameters(method.Operation),
			Responses:                spec.apiResponses(method.Operation),
		})
	}

	for _, webhook := range spec.apiWebhooks() {
		reference.Webhooks = append(reference.Webhooks, APIWebhook{
			Name:            webhook.Name,
			Anchor:          webhookAnchor(webhook),
			Description:     webhook.Operation.Description,
			RequestPayloads: spec.apiWebhookPayloads(webhook.Operation),
		})
	}

	for _, name := range spec.apiSchemaNames() {
		schema, ok := spec.Components.Schemas[name]
		if !ok {
			continue
		}
		reference.Types = append(reference.Types, APIType{
			Name:        name,
			Description: schema.Description,
			ExampleJSON: firstSchemaExampleJSON(schema.Examples),
			SchemaRows:  spec.schemaRows(schema),
		})
	}

	return reference
}

func renderAPIReferenceMarkdown(reference *APIReference, renderer goldmark.Markdown) error {
	for i := range reference.Methods {
		method := &reference.Methods[i]
		descriptionHTML, err := renderOpenAPIMarkdownHTML(renderer, method.Description)
		if err != nil {
			return fmt.Errorf("method %q description: %w", method.Name, err)
		}
		method.DescriptionHTML = descriptionHTML

		for j := range method.Parameters {
			param := &method.Parameters[j]
			descriptionHTML, err := renderOpenAPIMarkdownHTML(renderer, param.Description)
			if err != nil {
				return fmt.Errorf("method %q parameter %q description: %w", method.Name, param.Name, err)
			}
			param.DescriptionHTML = descriptionHTML
			if err := renderAPISchemaMarkdown(renderer, &param.Schema); err != nil {
				return fmt.Errorf("method %q parameter %q schema: %w", method.Name, param.Name, err)
			}
		}

		for j := range method.Responses {
			response := &method.Responses[j]
			descriptionHTML, err := renderOpenAPIMarkdownHTML(renderer, response.Description)
			if err != nil {
				return fmt.Errorf("method %q response %q description: %w", method.Name, response.Status, err)
			}
			response.DescriptionHTML = descriptionHTML
			if err := renderAPISchemaRowsMarkdown(renderer, response.SchemaRows); err != nil {
				return fmt.Errorf("method %q response %q schema: %w", method.Name, response.Status, err)
			}
		}
	}

	for i := range reference.Webhooks {
		webhook := &reference.Webhooks[i]
		descriptionHTML, err := renderOpenAPIMarkdownHTML(renderer, webhook.Description)
		if err != nil {
			return fmt.Errorf("webhook %q description: %w", webhook.Name, err)
		}
		webhook.DescriptionHTML = descriptionHTML

		for j := range webhook.RequestPayloads {
			payload := &webhook.RequestPayloads[j]
			descriptionHTML, err := renderOpenAPIMarkdownHTML(renderer, payload.Description)
			if err != nil {
				return fmt.Errorf("webhook %q payload %q description: %w", webhook.Name, payload.Name, err)
			}
			payload.DescriptionHTML = descriptionHTML
			if err := renderAPISchemaRowsMarkdown(renderer, payload.SchemaRows); err != nil {
				return fmt.Errorf("webhook %q payload %q schema: %w", webhook.Name, payload.Name, err)
			}
		}
	}

	for i := range reference.Types {
		apiType := &reference.Types[i]
		descriptionHTML, err := renderOpenAPIMarkdownHTML(renderer, apiType.Description)
		if err != nil {
			return fmt.Errorf("type %q description: %w", apiType.Name, err)
		}
		apiType.DescriptionHTML = descriptionHTML
		if err := renderAPISchemaRowsMarkdown(renderer, apiType.SchemaRows); err != nil {
			return fmt.Errorf("type %q schema: %w", apiType.Name, err)
		}
	}

	return nil
}

func renderOpenAPIMarkdownHTML(renderer goldmark.Markdown, markdown string) (string, error) {
	if strings.TrimSpace(markdown) == "" {
		return "", nil
	}
	contentHTML, _, err := renderMarkdownHTML(renderer, markdown)
	return contentHTML, err
}

func renderAPISchemaRowsMarkdown(renderer goldmark.Markdown, rows []APISchemaRow) error {
	for i := range rows {
		if err := renderAPISchemaMarkdown(renderer, &rows[i].Schema); err != nil {
			return fmt.Errorf("field %q: %w", rows[i].Field, err)
		}
	}
	return nil
}

func renderAPISchemaMarkdown(renderer goldmark.Markdown, schema *APISchema) error {
	descriptionHTML, err := renderOpenAPIMarkdownHTML(renderer, schema.Description)
	if err != nil {
		return fmt.Errorf("description: %w", err)
	}
	schema.DescriptionHTML = descriptionHTML

	if schema.Items != nil {
		if err := renderAPISchemaMarkdown(renderer, schema.Items); err != nil {
			return fmt.Errorf("items: %w", err)
		}
	}
	return nil
}

func (spec openAPISpec) apiParameters(op openAPIOperation) []APIParameter {
	params := make([]APIParameter, 0, len(op.Parameters))
	for _, param := range op.Parameters {
		params = append(params, APIParameter{
			Name:        param.Name,
			In:          param.In,
			Required:    param.Required,
			Description: param.Description,
			Schema:      spec.apiSchema(param.Schema),
		})
	}
	return params
}

func (spec openAPISpec) apiResponses(op openAPIOperation) []APIResponse {
	statuses := openAPIResponseStatuses(op)
	responses := make([]APIResponse, 0, len(statuses))
	for _, status := range statuses {
		response := spec.resolveResponse(op.Responses[status])
		apiResponse := APIResponse{
			Status:      status,
			Description: response.Description,
		}

		if strings.HasPrefix(status, "2") {
			if media, ok := response.Content["application/json"]; ok {
				if example, ok := firstExampleJSON(media.Examples); ok {
					apiResponse.ExampleJSON = example
				}
				apiResponse.SchemaRows = spec.schemaRows(media.Schema)
			}
		}

		responses = append(responses, apiResponse)
	}
	return responses
}

func (spec openAPISpec) apiSchemaNames() []string {
	excluded := spec.responseSchemaNames()
	for name := range spec.webhookPayloadSchemaNames() {
		excluded[name] = true
	}

	seen := map[string]bool{}
	names := []string{}
	var addSchema func(openAPISchemaRef, bool)
	addSchema = func(schema openAPISchemaRef, includeRef bool) {
		if schema.Ref != "" {
			name := refName(schema.Ref)
			if includeRef && !excluded[name] && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
			if resolved, ok := spec.resolveSchema(schema.Ref); ok {
				addSchema(resolved, true)
			}
			return
		}
		for _, part := range schema.AllOf {
			addSchema(part, includeRef)
		}
		for _, part := range schema.OneOf {
			addSchema(part, includeRef)
		}
		for _, propertyName := range orderedPropertyNames(schema) {
			addSchema(schema.Properties[propertyName], true)
		}
		if schema.Items != nil {
			addSchema(*schema.Items, true)
		}
	}

	for _, method := range spec.apiMethods() {
		for _, status := range openAPIResponseStatuses(method.Operation) {
			response := spec.resolveResponse(method.Operation.Responses[status])
			if media, ok := response.Content["application/json"]; ok {
				addSchema(media.Schema, false)
			}
		}
	}

	for _, webhook := range spec.apiWebhooks() {
		if media, ok := webhook.Operation.RequestBody.Content["application/json"]; ok {
			addSchema(media.Schema, true)
		}
	}

	return names
}

func (spec openAPISpec) webhookPayloadSchemaNames() map[string]bool {
	names := map[string]bool{}
	for _, webhook := range spec.apiWebhooks() {
		media, ok := webhook.Operation.RequestBody.Content["application/json"]
		if !ok {
			continue
		}
		for _, schema := range media.Schema.OneOf {
			if schema.Ref != "" {
				names[refName(schema.Ref)] = true
			}
		}
		if media.Schema.Ref != "" {
			names[refName(media.Schema.Ref)] = true
		}
	}
	return names
}

func (spec openAPISpec) responseSchemaNames() map[string]bool {
	names := map[string]bool{}
	for _, path := range spec.Paths {
		for _, op := range path {
			for _, response := range op.Responses {
				response = spec.resolveResponse(response)
				media, ok := response.Content["application/json"]
				if !ok || media.Schema.Ref == "" {
					continue
				}
				names[refName(media.Schema.Ref)] = true
			}
		}
	}
	return names
}

var (
	pathOperationMethods    = []string{"get", "post", "put", "patch", "delete"}
	webhookOperationMethods = []string{"post", "put", "patch", "delete", "get"}
)

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func (spec openAPISpec) apiMethods() []openAPIMethod {
	methods := []openAPIMethod{}
	for _, path := range sortedKeys(spec.Paths) {
		operations := spec.Paths[path]
		for _, method := range pathOperationMethods {
			op, ok := operations[method]
			if !ok {
				continue
			}
			methods = append(methods, openAPIMethod{
				Name:      operationName(path, op),
				Path:      path,
				Method:    method,
				Operation: op,
			})
		}
	}
	return methods
}

func (spec openAPISpec) apiWebhooks() []openAPIWebhook {
	webhooks := []openAPIWebhook{}
	for _, key := range sortedKeys(spec.Webhooks) {
		operations := spec.Webhooks[key]
		for _, method := range webhookOperationMethods {
			op, ok := operations[method]
			if !ok {
				continue
			}
			webhooks = append(webhooks, openAPIWebhook{
				Name:      operationName(key, op),
				Key:       key,
				Method:    method,
				Operation: op,
			})
		}
	}
	return webhooks
}

func operationName(path string, op openAPIOperation) string {
	if op.Summary != "" {
		return op.Summary
	}
	if op.OperationID != "" {
		return op.OperationID
	}
	return strings.TrimPrefix(path, "/")
}

func operationAnchor(method openAPIMethod) string {
	if method.Operation.OperationID != "" {
		return method.Operation.OperationID
	}
	return method.Name
}

func webhookAnchor(webhook openAPIWebhook) string {
	if webhook.Operation.OperationID != "" {
		return webhook.Operation.OperationID
	}
	return webhook.Name
}

func (spec openAPISpec) endpointPath(path string) string {
	serverURL := ""
	if len(spec.Servers) > 0 {
		serverURL = spec.Servers[0].URL
	}
	if serverURL == "" {
		return path
	}
	return strings.TrimRight(serverURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func (spec openAPISpec) operationAuthorization(op openAPIOperation) (string, string) {
	for _, security := range op.Security {
		if _, ok := security["bearerAuth"]; ok {
			return "Bearer token", spec.Components.SecuritySchemes["bearerAuth"].Description
		}
	}
	return "None", ""
}

func (spec openAPISpec) resolveResponse(response openAPIResponseRef) openAPIResponseRef {
	if response.Ref == "" {
		return response
	}
	name := refName(response.Ref)
	resolved, ok := spec.Components.Responses[name]
	if !ok {
		return response
	}
	return resolved
}

func (spec openAPISpec) apiWebhookPayloads(op openAPIOperation) []APIWebhookPayload {
	media, ok := op.RequestBody.Content["application/json"]
	if !ok {
		return nil
	}

	schemas := media.Schema.OneOf
	if len(schemas) == 0 {
		schemas = []openAPISchemaRef{media.Schema}
	}

	payloads := make([]APIWebhookPayload, 0, len(schemas))
	for _, schema := range schemas {
		name := schemaDisplayName(schema)
		description := schema.Description
		examples := schema.Examples
		if schema.Ref != "" {
			if resolved, ok := spec.resolveSchema(schema.Ref); ok {
				if description == "" {
					description = resolved.Description
				}
				examples = resolved.Examples
			}
		}

		payloads = append(payloads, APIWebhookPayload{
			Name:        name,
			Description: description,
			ExampleJSON: firstSchemaExampleJSON(examples),
			SchemaRows:  spec.schemaRows(schema),
		})
	}
	return payloads
}

func schemaDisplayName(schema openAPISchemaRef) string {
	if ref := effectiveRef(schema); ref != "" {
		return refName(ref)
	}
	return "Payload"
}

func (spec openAPISpec) schemaRows(schema openAPISchemaRef) []APISchemaRow {
	rows := []APISchemaRow{}
	spec.appendSchemaRows(&rows, "", schema, true, true)
	return rows
}

func (spec openAPISpec) appendSchemaRows(rows *[]APISchemaRow, path string, schema openAPISchemaRef, required bool, root bool) {
	if schema.Ref != "" {
		if !root {
			return
		}
		resolved, ok := spec.resolveSchema(schema.Ref)
		if !ok {
			return
		}
		spec.appendSchemaRows(rows, path, resolved, required, true)
		return
	}

	if len(schema.AllOf) > 0 {
		if !root && len(schema.AllOf) == 1 && schema.AllOf[0].Ref != "" {
			return
		}
		for _, part := range schema.AllOf {
			spec.appendSchemaRows(rows, path, part, required, true)
		}
		return
	}

	if schema.Type == "array" && schema.Items != nil {
		spec.appendSchemaRows(rows, path+"[]", *schema.Items, true, false)
		return
	}

	if schema.Type != "object" && len(schema.Properties) == 0 {
		return
	}

	requiredFields := make(map[string]bool, len(schema.Required))
	for _, field := range schema.Required {
		requiredFields[field] = true
	}

	for _, name := range orderedPropertyNames(schema) {
		property := schema.Properties[name]
		fieldPath := name
		if path != "" {
			fieldPath = path + "." + name
		}

		*rows = append(*rows, APISchemaRow{
			Field:    fieldPath,
			Schema:   spec.apiSchema(property),
			Required: requiredFields[name],
		})
		spec.appendSchemaRows(rows, fieldPath, property, requiredFields[name], false)
	}
}

func (spec openAPISpec) resolveSchema(ref string) (openAPISchemaRef, bool) {
	resolved, ok := spec.Components.Schemas[refName(ref)]
	return resolved, ok
}

func (spec openAPISpec) apiSchema(schema openAPISchemaRef) APISchema {
	apiSchema := APISchema{
		Type:        schema.Type,
		Description: schema.Description,
		Nullable:    schema.Nullable,
		AllOf:       len(schema.AllOf) > 0,
		Object:      len(schema.Properties) > 0,
		Format:      schema.Format,
		EnumValues:  apiSchemaEnumValues(schema.Enum),
		MaxItems:    schema.MaxItems,
		MinItems:    schema.MinItems,
	}
	if ref := effectiveRef(schema); ref != "" {
		name := refName(ref)
		apiSchema.RefName = name
		apiSchema.RefURL = apiReferenceURL(name)
		if apiSchema.Description == "" {
			if resolved, ok := spec.resolveSchema(ref); ok {
				apiSchema.Description = resolved.Description
			}
		}
	}
	if schema.Items != nil {
		itemSchema := spec.apiSchema(*schema.Items)
		apiSchema.Items = &itemSchema
	}
	if schema.Const != nil {
		apiSchema.HasConst = true
		apiSchema.ConstType = fmt.Sprintf("%T", schema.Const)
		apiSchema.ConstValue = fmt.Sprint(schema.Const)
	}
	return apiSchema
}

// effectiveRef returns the ref string the schema effectively points to.
// This handles both a direct $ref and the common "wrap a $ref in a single
// allOf to attach a description" pattern.
func effectiveRef(schema openAPISchemaRef) string {
	if schema.Ref != "" {
		return schema.Ref
	}
	if len(schema.AllOf) == 1 && schema.AllOf[0].Ref != "" {
		return schema.AllOf[0].Ref
	}
	return ""
}

func apiSchemaEnumValues(values []any) []string {
	if len(values) == 0 {
		return nil
	}

	enumValues := make([]string, len(values))
	for i, value := range values {
		enumValues[i] = fmt.Sprint(value)
	}
	return enumValues
}

func orderedPropertyNames(schema openAPISchemaRef) []string {
	names := make([]string, 0, len(schema.Properties))
	seen := make(map[string]bool, len(schema.Properties))
	for _, name := range schema.Required {
		if _, ok := schema.Properties[name]; ok {
			names = append(names, name)
			seen[name] = true
		}
	}

	remaining := make([]string, 0, len(schema.Properties)-len(names))
	for name := range schema.Properties {
		if !seen[name] {
			remaining = append(remaining, name)
		}
	}
	slices.Sort(remaining)
	return append(names, remaining...)
}

func firstExampleJSON(examples map[string]openAPIExampleRef) (string, bool) {
	if len(examples) == 0 {
		return "", false
	}
	first := sortedKeys(examples)[0]
	return exampleJSON(examples[first].Value)
}

func firstSchemaExampleJSON(examples []yaml.Node) string {
	if len(examples) == 0 {
		return ""
	}
	jsonValue, _ := exampleJSON(examples[0])
	return jsonValue
}

func exampleJSON(value yaml.Node) (string, bool) {
	var buf bytes.Buffer
	if err := writeYAMLNodeJSON(&buf, value, 0); err != nil {
		return "", false
	}
	return buf.String(), true
}

func writeYAMLNodeJSON(buf *bytes.Buffer, node yaml.Node, depth int) error {
	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			buf.WriteString("null")
			return nil
		}
		return writeYAMLNodeJSON(buf, *node.Content[0], depth)
	case yaml.MappingNode:
		if len(node.Content) == 0 {
			buf.WriteString("{}")
			return nil
		}
		buf.WriteString("{\n")
		for i := 0; i < len(node.Content); i += 2 {
			if i > 0 {
				buf.WriteString(",\n")
			}
			writeJSONIndent(buf, depth+1)
			key, err := yamlMappingKeyString(*node.Content[i])
			if err != nil {
				return err
			}
			jsonKey, err := jsonMarshalNoEscape(key)
			if err != nil {
				return err
			}
			buf.Write(jsonKey)
			buf.WriteString(": ")
			if err := writeYAMLNodeJSON(buf, *node.Content[i+1], depth+1); err != nil {
				return err
			}
		}
		buf.WriteByte('\n')
		writeJSONIndent(buf, depth)
		buf.WriteByte('}')
		return nil
	case yaml.SequenceNode:
		if len(node.Content) == 0 {
			buf.WriteString("[]")
			return nil
		}
		buf.WriteString("[\n")
		for i, item := range node.Content {
			if i > 0 {
				buf.WriteString(",\n")
			}
			writeJSONIndent(buf, depth+1)
			if err := writeYAMLNodeJSON(buf, *item, depth+1); err != nil {
				return err
			}
		}
		buf.WriteByte('\n')
		writeJSONIndent(buf, depth)
		buf.WriteByte(']')
		return nil
	default:
		var value any
		if err := node.Decode(&value); err != nil {
			return err
		}
		jsonValue, err := jsonMarshalNoEscape(value)
		if err != nil {
			return err
		}
		buf.Write(jsonValue)
		return nil
	}
}

func jsonMarshalNoEscape(value any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func yamlMappingKeyString(node yaml.Node) (string, error) {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		return node.Value, nil
	}

	var key any
	if err := node.Decode(&key); err != nil {
		return "", err
	}
	return fmt.Sprint(key), nil
}

func writeJSONIndent(buf *bytes.Buffer, depth int) {
	buf.WriteString(strings.Repeat("  ", depth))
}

func refName(ref string) string {
	ref = strings.TrimSpace(ref)
	if idx := strings.LastIndex(ref, "/"); idx != -1 {
		return ref[idx+1:]
	}
	return ref
}

func apiReferenceURL(name string) string {
	return apiReferencePath + "#" + strings.TrimSpace(name)
}

func openAPIResponseStatuses(op openAPIOperation) []string {
	return sortedKeys(op.Responses)
}
