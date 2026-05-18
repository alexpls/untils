package docs

import "strconv"

type APIReference struct {
	Methods  []APIMethod
	Webhooks []APIWebhook
	Types    []APIType
}

type APIMethod struct {
	Name                     string
	Anchor                   string
	Description              string
	DescriptionHTML          string
	HTTPMethod               string
	Endpoint                 string
	Authorization            string
	AuthorizationDescription string
	Parameters               []APIParameter
	Responses                []APIResponse
}

type APIParameter struct {
	Name            string
	In              string
	Required        bool
	Description     string
	DescriptionHTML string
	Schema          APISchema
}

type APIResponse struct {
	Status          string
	Description     string
	DescriptionHTML string
	ExampleJSON     string
	SchemaRows      []APISchemaRow
}

type APIWebhook struct {
	Name            string
	Anchor          string
	Description     string
	DescriptionHTML string
	RequestPayloads []APIWebhookPayload
}

type APIWebhookPayload struct {
	Name            string
	Description     string
	DescriptionHTML string
	ExampleJSON     string
	SchemaRows      []APISchemaRow
}

type APIType struct {
	Name            string
	Description     string
	DescriptionHTML string
	ExampleJSON     string
	SchemaRows      []APISchemaRow
}

type APISchemaRow struct {
	Field    string
	Schema   APISchema
	Required bool
}

type APISchema struct {
	Description     string
	DescriptionHTML string
	RefName         string
	RefURL          string
	Type            string
	Nullable        bool
	Items           *APISchema
	AllOf           bool
	Object          bool
	HasConst        bool
	ConstType       string
	ConstValue      string
	Format          string
	EnumValues      []string
	MaxItems        *int
	MinItems        *int
}

func APIReferenceNav(reference APIReference) []NavPage {
	return []NavPage{
		{
			SidebarTitle: "Methods",
			Path:         apiReferenceURL("methods"),
			Children:     apiMethodNav(reference.Methods),
		},
		{
			SidebarTitle: "Webhooks",
			Path:         apiReferenceURL("webhooks"),
			Children:     apiWebhookNav(reference.Webhooks),
		},
		{
			SidebarTitle: "Types",
			Path:         apiReferenceURL("types"),
			Children:     apiTypeNav(reference.Types),
		},
	}
}

func stringFromInt(value int) string {
	return strconv.Itoa(value)
}

func apiMethodNav(methods []APIMethod) []NavPage {
	pages := make([]NavPage, 0, len(methods))
	for _, method := range methods {
		pages = append(pages, NavPage{
			SidebarTitle: method.Name,
			Path:         apiReferenceURL(method.Anchor),
		})
	}
	return pages
}

func apiWebhookNav(webhooks []APIWebhook) []NavPage {
	pages := []NavPage{}
	for _, webhook := range webhooks {
		for _, payload := range webhook.RequestPayloads {
			pages = append(pages, NavPage{
				SidebarTitle: payload.Name,
				Path:         apiReferenceURL(payload.Name),
			})
		}
	}
	return pages
}

func apiTypeNav(types []APIType) []NavPage {
	pages := make([]NavPage, 0, len(types))
	for _, apiType := range types {
		pages = append(pages, NavPage{
			SidebarTitle: apiType.Name,
			Path:         apiReferenceURL(apiType.Name),
		})
	}
	return pages
}
