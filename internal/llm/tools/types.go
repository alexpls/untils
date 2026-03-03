package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/alexpls/untils/internal/browser"
	"github.com/alexpls/untils/internal/search"
)

// Definition describes a tool exposed to the LLM.
type Definition struct {
	Name         string
	Description  string
	UsageHeading string
	UsageBody    string
	Params       any
}

// Context contains runtime dependencies for tool execution and validation.
type Context struct {
	Ctx             context.Context
	Logger          *slog.Logger
	Browser         func() *browser.BrowserCtx
	Search          func(params *search.SearchParams) (*search.SearchResponse, error)
	ReadInstruction func(name string) (string, error)
	AddSiteVisited  func(url string)
	PriorCalls      *[]Call
}

// Call holds a prepared tool call with parsed parameters.
type Call struct {
	call     func() (string, error)
	validate func() string
	params   any
}

func (c *Call) Execute() (string, error) {
	return c.call()
}

func (c *Call) Validate() string {
	return c.validate()
}

func (c *Call) Params() any {
	return c.params
}

type Builder func(tc *Context, args string) (*Call, error)

type Tool interface {
	Definition() Definition
	Builder() Builder
}

type tool[P any] struct {
	name         string
	description  string
	usageHeading string
	usageBody    string
	execute      func(tc *Context, params P) (string, error)
	validate     func(tc *Context, params P) string
}

func (t tool[P]) Definition() Definition {
	var zero P
	return Definition{
		Name:         t.name,
		Description:  t.description,
		UsageHeading: t.usageHeading,
		UsageBody:    t.usageBody,
		Params:       zero,
	}
}

func (t tool[P]) Builder() Builder {
	return func(tc *Context, args string) (*Call, error) {
		var params P
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return nil, fmt.Errorf("unmarshaling %s params: %w", t.name, err)
		}
		return &Call{
			call:     func() (string, error) { return t.execute(tc, params) },
			validate: func() string { return t.validate(tc, params) },
			params:   params,
		}, nil
	}
}
