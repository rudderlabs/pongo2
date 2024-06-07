package pongo2

import (
	"fmt"
	"regexp"
)

var reIdentifiers = regexp.MustCompile("^[a-zA-Z0-9_]+$")

var autoescape = true

func SetAutoescape(newValue bool) {
	autoescape = newValue
}

// A Context type provides constants, variables, instances or functions to a template.
//
// pongo2 automatically provides meta-information or functions through the "pongo2"-key.
// Currently, context["pongo2"] contains the following keys:
//  1. version: returns the version string
//
// Template examples for accessing items from your context:
//
//	{{ myconstant }}
//	{{ myfunc("test", 42) }}
//	{{ user.name }}
//	{{ pongo2.version }}
type Context struct {
	*SyncMap[string, any]
}

func NewContext() *Context {
	return &Context{
		SyncMap: NewSyncMap[string, any](),
	}
}

func (c *Context) checkForValidIdentifiers() *Error {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for k, v := range c.m {
		if !reIdentifiers.MatchString(k) {
			return &Error{
				Sender:    "checkForValidIdentifiers",
				OrigError: fmt.Errorf("context-key '%s' (value: '%+v') is not a valid identifier", k, v),
			}
		}
	}
	return nil
}

// Update updates this context with the key/value-pairs from another context.
func (c *Context) Update(other *Context) *Context {
	other.lock.RLock()
	defer other.lock.RUnlock()

	c.lock.Lock()
	defer c.lock.Unlock()

	for k, v := range other.m {
		c.m[k] = v
	}
	return c
}

func (c *Context) Length() int {
	return len(c.m)
}

// ExecutionContext contains all data important for the current rendering state.
//
// If you're writing a custom tag, your tag's Execute()-function will
// have access to the ExecutionContext. This struct stores anything
// about the current rendering process's Context including
// the Context provided by the user (field Public).
// You can safely use the Private context to provide data to the user's
// template (like a 'forloop'-information). The Shared-context is used
// to share data between tags. All ExecutionContexts share this context.
//
// Please be careful when accessing the Public data.
// PLEASE DO NOT MODIFY THE PUBLIC CONTEXT (read-only).
//
// To create your own execution context within tags, use the
// NewChildExecutionContext(parent) function.
type ExecutionContext struct {
	template   *Template
	macroDepth int

	AllowMissingVal bool
	Autoescape      bool
	Public          *Context
	Private         *Context
	Shared          *Context
}


func newExecutionContext(tpl *Template, ctx *Context) *ExecutionContext {
	privateCtx := NewContext()

	// Make the pongo2-related funcs/vars available to the context
	// No need to lock privateCtx, as it is not yet shared
	privateCtx.Set("pongo2", map[string]any{"version": Version})

	ctx.lock.Lock()
	defer ctx.lock.Unlock()	
	ctx.m["nil"] = nil

	return &ExecutionContext{
		template: tpl,

		Public:     ctx,
		Private:    privateCtx,
		Autoescape: autoescape,
	}
}

func NewChildExecutionContext(parent *ExecutionContext) *ExecutionContext {
	newctx := &ExecutionContext{
		template: parent.template,

		Public:     parent.Public,
		Private:    NewContext(),
		Autoescape: parent.Autoescape,
	}
	if parent.Shared != nil {
		parent.Shared.lock.Lock()
		defer parent.Shared.lock.Unlock()
	}
	// No need to lock newctx, as it is not yet shared
	newctx.Shared = parent.Shared

	// Copy all existing private items
	newctx.Private.Update(parent.Private)

	return newctx
}

func (ctx *ExecutionContext) Error(err error, token *Token) *Error {
	return ctx.OrigError(err, token)
}

func (ctx *ExecutionContext) OrigError(err error, token *Token) *Error {
	filename := ctx.template.name
	var line, col int
	if token != nil {
		// No tokens available
		// TODO: Add location (from where?)
		filename = token.Filename
		line = token.Line
		col = token.Col
	}
	return &Error{
		Template:  ctx.template,
		Filename:  filename,
		Line:      line,
		Column:    col,
		Token:     token,
		Sender:    "execution",
		OrigError: err,
	}
}

func (ctx *ExecutionContext) Logf(format string, args ...any) {
	ctx.template.set.logf(format, args...)
}
