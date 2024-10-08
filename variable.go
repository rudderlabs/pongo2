package pongo2

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	varTypeInt = iota
	varTypeIdent
	varTypeAttr
	varTypeSubscript
	varTypeArray
	varTypeNil

	getAttrMethodName = "GetAttr"
)

var (
	typeOfValuePtr   = reflect.TypeOf(new(Value))
	typeOfExecCtxPtr = reflect.TypeOf(new(ExecutionContext))
	typeOfKwargsMap  = reflect.TypeOf(make(map[string]*Value))
)

type variablePart struct {
	typ       int
	s         string
	i         int
	subscript IEvaluator
	isNil     bool

	isFunctionCall bool
	callingArgs    []functionCallArgument // needed for a function call, represents all argument nodes (INode supports nested function calls)
}

func (p *variablePart) String() string {
	switch p.typ {
	case varTypeInt:
		return strconv.Itoa(p.i)
	case varTypeIdent:
		return p.s
	case varTypeAttr:
		return "@" + p.s + "{attr args}"
	case varTypeSubscript:
		return "[subscript]"
	case varTypeArray:
		return "[array]"
	}

	panic("unimplemented")
}

type functionCallArgument interface {
	Evaluate(*ExecutionContext) (*Value, *Error)
}

// TODO: Add location tokens
type stringResolver struct {
	locationToken *Token
	val           string
}

type intResolver struct {
	locationToken *Token
	val           int
}

type floatResolver struct {
	locationToken *Token
	val           float64
}

type boolResolver struct {
	locationToken *Token
	val           bool
}

type variableResolver struct {
	locationToken *Token

	parts []*variablePart
}

type nodeFilteredVariable struct {
	locationToken *Token

	resolver    IEvaluator
	filterChain []*filterCall
}

type nodeVariable struct {
	locationToken *Token
	expr          IEvaluator
}

type executionCtxEval struct{}

func (v *nodeFilteredVariable) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	value, err := v.Evaluate(ctx)
	if err != nil {
		return err
	}
	if _, err := writer.WriteString(value.String()); err != nil {
		return ctx.Error(err, v.locationToken)
	}
	return nil
}

func (vr *variableResolver) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	value, err := vr.Evaluate(ctx)
	if err != nil {
		return err
	}
	if _, err := writer.WriteString(value.String()); err != nil {
		return ctx.Error(err, vr.locationToken)
	}
	return nil
}

func (s *stringResolver) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	value, err := s.Evaluate(ctx)
	if err != nil {
		return err
	}
	if _, err := writer.WriteString(value.String()); err != nil {
		return ctx.Error(err, s.locationToken)
	}
	return nil
}

func (i *intResolver) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	value, err := i.Evaluate(ctx)
	if err != nil {
		return err
	}
	if _, err := writer.WriteString(value.String()); err != nil {
		return ctx.Error(err, i.locationToken)
	}
	return nil
}

func (f *floatResolver) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	value, err := f.Evaluate(ctx)
	if err != nil {
		return err
	}
	if _, err := writer.WriteString(value.String()); err != nil {
		return ctx.Error(err, f.locationToken)
	}
	return nil
}

func (b *boolResolver) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	value, err := b.Evaluate(ctx)
	if err != nil {
		return err
	}
	if _, err := writer.WriteString(value.String()); err != nil {
		return ctx.Error(err, b.locationToken)
	}
	return nil
}

func (v *nodeFilteredVariable) GetPositionToken() *Token {
	return v.locationToken
}

func (vr *variableResolver) GetPositionToken() *Token {
	return vr.locationToken
}

func (s *stringResolver) GetPositionToken() *Token {
	return s.locationToken
}

func (i *intResolver) GetPositionToken() *Token {
	return i.locationToken
}

func (f *floatResolver) GetPositionToken() *Token {
	return f.locationToken
}

func (b *boolResolver) GetPositionToken() *Token {
	return b.locationToken
}

func (s *stringResolver) Evaluate(ctx *ExecutionContext) (*Value, *Error) {
	return AsValue(s.val), nil
}

func (i *intResolver) Evaluate(ctx *ExecutionContext) (*Value, *Error) {
	return AsValue(i.val), nil
}

func (f *floatResolver) Evaluate(ctx *ExecutionContext) (*Value, *Error) {
	return AsValue(f.val), nil
}

func (b *boolResolver) Evaluate(ctx *ExecutionContext) (*Value, *Error) {
	return AsValue(b.val), nil
}

func (s *stringResolver) FilterApplied(name string) bool {
	return false
}

func (i *intResolver) FilterApplied(name string) bool {
	return false
}

func (f *floatResolver) FilterApplied(name string) bool {
	return false
}

func (b *boolResolver) FilterApplied(name string) bool {
	return false
}

func (nv *nodeVariable) FilterApplied(name string) bool {
	return nv.expr.FilterApplied(name)
}

func (nv *nodeVariable) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	value, err := nv.expr.Evaluate(ctx)
	if err != nil {
		return err
	}

	if !nv.expr.FilterApplied("safe") && !value.safe && value.IsString() && ctx.Autoescape {
		// apply escape filter
		value, err = filters["escape"](value, nil)
		if err != nil {
			return err
		}
	}

	if _, err := writer.WriteString(value.String()); err != nil {
		return ctx.Error(err, nv.locationToken)
	}
	return nil
}

func (executionCtxEval) Evaluate(ctx *ExecutionContext) (*Value, *Error) {
	return AsValue(ctx), nil
}

func (vr *variableResolver) FilterApplied(name string) bool {
	return false
}

func (vr *variableResolver) String() string {
	parts := make([]string, 0, len(vr.parts))
	for _, p := range vr.parts {
		parts = append(parts, p.String())
	}

	return strings.Join(parts, ".")
}

func (vr *variableResolver) resolve(ctx *ExecutionContext) (*Value, error) {
	var current reflect.Value
	var currentPresent bool
	var isSafe bool

	// we are resolving an in-template array definition
	if len(vr.parts) > 0 && vr.parts[0].typ == varTypeArray {
		items := make([]*Value, 0)
		for _, part := range vr.parts {
			switch v := part.subscript.(type) {
			case *nodeFilteredVariable:
				item, err := v.resolver.Evaluate(ctx)
				if err != nil {
					return nil, err
				}

				items = append(items, item)
			default:
				return nil, errors.New("unknown variable type is given")
			}
		}

		return &Value{
			val:  reflect.ValueOf(items),
			safe: true,
		}, nil
	}

	for idx, part := range vr.parts {
		currentPresent = false
		assumeAttr := false
		if idx == 0 {
			// We're looking up the first part of the variable.
			// First we're having a look in our private
			// context (e. g. information provided by tags, like the forloop)
			val, inPrivate := ctx.Private[vr.parts[0].s]

			if !inPrivate {
				// Nothing found? Then have a final lookup in the public context
				val, currentPresent = ctx.Public[vr.parts[0].s]
			}

			current = reflect.ValueOf(val) // Get the initial value

		} else {
			// Next parts, resolve it from current

			// Before resolving the pointer, let's see if we have a method to call
			// Problem with resolving the pointer is we're changing the receiver
			isFunc := false
			funcName := ""
			if part.typ == varTypeIdent {
				funcName = part.s
			} else if part.typ == varTypeAttr {
				assumeAttr = true
				funcName = getAttrMethodName
			}
			if funcName != "" {
				funcValue := current.MethodByName(funcName)
				if funcValue.IsValid() {
					current = funcValue
					currentPresent = true
					isFunc = true
				} else if part.typ == varTypeAttr {
					return nil, fmt.Errorf("can't access method %s on type %s (variable %s)",
						funcName, current.Kind().String(), vr.String())
				}
			}

			if !isFunc {
				// If current a pointer, resolve it
				if current.Kind() == reflect.Ptr {
					current = current.Elem()
					if !current.IsValid() {
						// Value is not valid (anymore)
						return AsValue(nil), nil
					}
				}

				// Look up which part must be called now
				switch part.typ {
				case varTypeInt:
					// Calling an index is only possible for:
					// * slices/arrays/strings
					switch current.Kind() {
					case reflect.String, reflect.Array, reflect.Slice:
						if part.i >= 0 && current.Len() > part.i {
							current = current.Index(part.i)
							currentPresent = true
						} else {
							// In Django, exceeding the length of a list is just empty.
							return AsValue(nil), nil
						}
					default:
						return nil, fmt.Errorf("can't access an index on type %s (variable %s)",
							current.Kind().String(), vr.String())
					}
				case varTypeIdent:
					var tryField reflect.Value
					// Calling a field or key
					switch current.Kind() {
					case reflect.Struct:
						tryField = current.FieldByName(part.s)
					case reflect.Map:
						tryField = current.MapIndex(reflect.ValueOf(part.s))
					default:
						return nil, fmt.Errorf("can't access a field by name on type %s (variable %s)",
							current.Kind().String(), vr.String())
					}
					if tryField.IsValid() {
						current = tryField
					} else {
						getAttr := current.MethodByName(getAttrMethodName)
						if !getAttr.IsValid() && current.CanAddr() {
							getAttr = current.Addr().MethodByName(getAttrMethodName)
						}
						if getAttr.IsValid() {
							current = getAttr
							currentPresent = true
							assumeAttr = true
						} else {
							current = tryField
						}
					}
				case varTypeSubscript:
					// Calling an index is only possible for:
					// * slices/arrays/strings
					switch current.Kind() {
					case reflect.String, reflect.Array, reflect.Slice:
						sv, err := part.subscript.Evaluate(ctx)
						if err != nil {
							return nil, err
						}
						si := sv.Integer()
						if si >= 0 && current.Len() > si {
							current = current.Index(si)
						} else {
							// In Django, exceeding the length of a list is just empty.
							return AsValue(nil), nil
						}
					// Calling a field or key
					case reflect.Struct:
						sv, err := part.subscript.Evaluate(ctx)
						if err != nil {
							return nil, err
						}
						current = current.FieldByName(sv.String())
						currentPresent = true
					case reflect.Map:
						sv, err := part.subscript.Evaluate(ctx)
						if err != nil {
							return nil, err
						}
						if sv.IsNil() {
							return AsValue(nil), nil
						}
						if sv.val.Type().AssignableTo(current.Type().Key()) {
							current = current.MapIndex(sv.val)
							currentPresent = true
						} else {
							return AsValue(nil), nil
						}
					default:
						return nil, fmt.Errorf("can't access an index on type %s (variable %s)",
							current.Kind().String(), vr.String())
					}
				default:
					panic("unimplemented")
				}
			}
		}

		if !current.IsValid() {
			// Value is not valid (anymore)
			if !currentPresent {
				if ctx.AllowMissingVal {
					return AsValue(nil), nil
				}

				return AsValue("NOT FOUND"), fmt.Errorf("No value found for %s", vr)
			}

			return AsValue(nil), nil

		}

		// If current is a reflect.ValueOf(pongo2.Value), then unpack it
		// Happens in function calls (as a return value) or by injecting
		// into the execution context (e.g. in a for-loop)
		if current.Type() == typeOfValuePtr {
			tmpValue := current.Interface().(*Value)
			current = tmpValue.val
			isSafe = tmpValue.safe
			currentPresent = true
		}

		// Check whether this is an interface and resolve it where required
		if current.Kind() == reflect.Interface {
			current = reflect.ValueOf(current.Interface())
			currentPresent = true
		}

		// Check if the part is a function call
		if part.isFunctionCall || current.Kind() == reflect.Func {
			// Check for callable
			if current.Kind() != reflect.Func {
				getAttr := current.MethodByName(getAttrMethodName)
				if getAttr.IsValid() {
					current = getAttr
					currentPresent = true
					assumeAttr = true
				} else {
					return nil, fmt.Errorf("'%s' is not a function (it is %s)", vr.String(), current.Kind().String())
				}
			}

			// Check for correct function syntax and types
			// func(*Value, ...) *Value
			t := current.Type()
			currArgs := part.callingArgs

			// If an implicit ExecCtx is needed
			if t.NumIn() > 0 && t.In(0) == typeOfExecCtxPtr {
				currArgs = append([]functionCallArgument{executionCtxEval{}}, currArgs...)
			}

			// If kwargs map is needed
			includeKwargsBit := 0
			if t.NumIn() > 0 && t.In(0) == typeOfKwargsMap {
				includeKwargsBit = 1
			}

			evaluatedArgs := make([]*Value, 0)
			kwargCount := 0
			for _, arg := range currArgs {
				v, err := arg.Evaluate(ctx)
				if err != nil {
					return nil, err
				}
				if v.IsKwarg() {
					if !(includeKwargsBit == 1) {
						return nil, fmt.Errorf("calling a function using a keyword argument: %v=%v, but the function does not support kwargs. Add map[string]*pongo2.Value as first arg in the function", v.Name(), v)
					}
					kwargCount++
				}
				evaluatedArgs = append(evaluatedArgs, v)
			}

			hasAttrNameArg := 0
			if assumeAttr {
				hasAttrNameArg = 1
			}

			// Input arguments
			if len(currArgs)+hasAttrNameArg-kwargCount != t.NumIn()-includeKwargsBit &&
				!(len(currArgs)+hasAttrNameArg-kwargCount >= t.NumIn()-1-includeKwargsBit &&
					t.IsVariadic()) {
				return nil,
					fmt.Errorf("function input argument count (%d) of '%s' must be equal to the calling argument count (%d)",
						t.NumIn()-includeKwargsBit, vr.String(), len(currArgs)-kwargCount)
			}

			// Output arguments
			if t.NumOut() != 1 && t.NumOut() != 2 {
				return nil, fmt.Errorf("'%s' must have exactly 1 or 2 output arguments, the second argument must be of type error", vr.String())
			}

			// Evaluate all parameters
			args := make([]reflect.Value, 0)
			kwargs := make(map[string]reflect.Value)

			if assumeAttr {
				var value any = part.s
				if includeKwargsBit == 1 {
					value = AsValue(part.s)
				}
				args = append(args, reflect.ValueOf(value))
			}

			numArgs := t.NumIn()
			isVariadic := t.IsVariadic()
			var fnArg reflect.Type

			for idx, pv := range evaluatedArgs {
				if pv.IsKwarg() {
					fnArg = typeOfValuePtr
				} else if isVariadic {
					if idx+includeKwargsBit >= t.NumIn()-1 {
						fnArg = t.In(numArgs - 1).Elem()
					} else {
						fnArg = t.In(idx + includeKwargsBit)
					}
				} else {
					fnArg = t.In(idx + includeKwargsBit)
				}

				var val reflect.Value
				if fnArg != typeOfValuePtr {
					// Function's argument is not a *pongo2.Value, then we have to check whether input argument is of the same type as the function's argument
					if !isVariadic {
						if fnArg != reflect.TypeOf(pv.Interface()) && fnArg.Kind() != reflect.Interface && fnArg.Kind() != reflect.ValueOf(kwargs).Kind() {
							return nil, fmt.Errorf("function input argument %d of '%s' must be of type %s or *pongo2.Value (not %T)",
								idx, vr.String(), fnArg.String(), pv.Interface())
						}
					} else {
						if fnArg != reflect.TypeOf(pv.Interface()) && fnArg.Kind() != reflect.Interface && fnArg.Kind() != reflect.ValueOf(kwargs).Kind() {
							return nil, fmt.Errorf("function variadic input argument of '%s' must be of type %s or *pongo2.Value (not %T)",
								vr.String(), fnArg.String(), pv.Interface())
						}
					}

					if pv.IsNil() {
						// Workaround to present an interface nil as reflect.Value
						var empty any = nil
						val = reflect.ValueOf(&empty).Elem()
					} else {
						val = reflect.ValueOf(pv.Interface())
					}
				} else {
					if pv.IsKwarg() {
						val = reflect.ValueOf(pv.Interface())
					} else {
						// Function's argument is a *pongo2.Value
						val = reflect.ValueOf(pv)
					}
				}

				if val.Kind() == reflect.Invalid {
					return nil, fmt.Errorf("calling a function using an invalid parameter")
				}

				if pv.IsKwarg() {
					kwargs[pv.Name()] = val
				} else if len(kwargs) == 0 {
					args = append(args, val)
				} else {
					return nil, fmt.Errorf("calling a function using a positional argument: %v, after a keyword argument", pv)
				}
			}

			pkwargs := make(map[string]*Value)
			for k, v := range kwargs {
				pv, ok := v.Interface().(*Value)
				if !ok {
					return nil, fmt.Errorf("calling a function using an invalid parameter")
				}
				pkwargs[k] = AsNamedValue(k, pv.Interface())
			}
			parameters := make([]reflect.Value, 0)
			if includeKwargsBit == 1 {
				parameters = append(parameters, reflect.ValueOf(pkwargs))
			}
			parameters = append(parameters, args...)

			for _, p := range parameters {
				if p.Kind() == reflect.Invalid {
					return nil, fmt.Errorf("calling a function using an invalid parameter")
				}
			}

			// Call it and get first return parameter back
			values := current.Call(parameters)
			rv := values[0]
			if t.NumOut() == 2 {
				e := values[1].Interface()
				if e != nil {
					err, ok := e.(error)
					if !ok {
						return nil, fmt.Errorf("the second return value is not an error")
					}
					if err != nil {
						return nil, err
					}
				}
			}

			if rv.Type() != typeOfValuePtr {
				current = reflect.ValueOf(rv.Interface())
			} else {
				// Return the function call value
				current = rv.Interface().(*Value).val
				isSafe = rv.Interface().(*Value).safe
			}
		}

		if !current.IsValid() {
			// Value is not valid (e. g. NIL value)
			if !currentPresent {

				if ctx.AllowMissingVal {
					return AsValue(nil), nil
				}

				return AsValue("NOT FOUND"), fmt.Errorf("No value found for %s", vr)
			}
			return AsValue(nil), nil
		}
	}

	return &Value{val: current, safe: isSafe}, nil
}

func (vr *variableResolver) Evaluate(ctx *ExecutionContext) (*Value, *Error) {
	value, err := vr.resolve(ctx)
	if err != nil {
		return AsValue(nil), ctx.Error(err, vr.locationToken)
	}
	return value, nil
}

func (v *nodeFilteredVariable) FilterApplied(name string) bool {
	for _, filter := range v.filterChain {
		if filter.name == name {
			return true
		}
	}
	return false
}

func (v *nodeFilteredVariable) Evaluate(ctx *ExecutionContext) (*Value, *Error) {
	value, err := v.resolver.Evaluate(ctx)
	if err != nil {
		return nil, err
	}

	for _, filter := range v.filterChain {
		value, err = filter.Execute(value, ctx)
		if err != nil {
			return nil, err
		}
	}

	return value, nil
}

// "[" [expr {, expr}] "]"
func (p *Parser) parseArray() (IEvaluator, *Error) {
	resolver := &variableResolver{
		locationToken: p.Current(),
	}
	p.Consume() // We consume '['

	// We allow an empty list, so check for a closing bracket.
	if p.Match(TokenSymbol, "]") != nil {
		return resolver, nil
	}

	// parsing an array declaration with at least one expression
	for {
		if p.Remaining() == 0 {
			return nil, p.Error(fmt.Errorf("Unexpected EOF, unclosed array list."), p.lastToken)
		}

		// No closing bracket, so we're parsing an expression
		exprArg, err := p.ParseExpression()
		if err != nil {
			return nil, err
		}

		resolver.parts = append(resolver.parts, &variablePart{
			typ:       varTypeArray,
			subscript: exprArg,
		})

		if p.Match(TokenSymbol, "]") != nil {
			// If there's a closing bracket after an expression, we will stop parsing the arguments
			break
		}

		// If there's NO closing bracket, there MUST be an comma
		if p.Match(TokenSymbol, ",") == nil {
			return nil, p.Error(fmt.Errorf("Missing comma or closing bracket after argument."), p.Current())
		}
	}

	return resolver, nil
}

// IDENT | IDENT.(IDENT|NUMBER)... | IDENT[expr]... | "[" [ expr {, expr}] "]"
func (p *Parser) parseVariableOrLiteral() (IEvaluator, *Error) {
	t := p.Current()

	if t == nil {
		return nil, p.Error(fmt.Errorf("Unexpected EOF, expected a number, string, keyword or identifier."), p.lastToken)
	}

	// Is first part a number or a string, there's nothing to resolve (because there's only to return the value then)
	switch t.Typ {
	case TokenNumber:
		p.Consume()

		// One exception to the rule that we don't have float64 literals is at the beginning
		// of an expression (or a variable name). Since we know we started with an integer
		// which can't obviously be a variable name, we can check whether the first number
		// is followed by dot (and then a number again). If so we're converting it to a float64.

		if p.Match(TokenSymbol, ".") != nil {
			// float64
			t2 := p.MatchType(TokenNumber)
			if t2 == nil {
				return nil, p.Error(fmt.Errorf("Expected a number after the '.'."), nil)
			}
			f, err := strconv.ParseFloat(fmt.Sprintf("%s.%s", t.Val, t2.Val), 64)
			if err != nil {
				return nil, p.Error(err, t)
			}
			fr := &floatResolver{
				locationToken: t,
				val:           f,
			}
			return fr, nil
		}
		i, err := strconv.Atoi(t.Val)
		if err != nil {
			return nil, p.Error(err, t)
		}
		nr := &intResolver{
			locationToken: t,
			val:           i,
		}
		return nr, nil
	case TokenString:
		p.Consume()
		sr := &stringResolver{
			locationToken: t,
			val:           t.Val,
		}
		return sr, nil
	case TokenKeyword:
		p.Consume()
		switch t.Val {
		case "true":
			br := &boolResolver{
				locationToken: t,
				val:           true,
			}
			return br, nil
		case "false":
			br := &boolResolver{
				locationToken: t,
				val:           false,
			}
			return br, nil
		default:
			return nil, p.Error(fmt.Errorf("This keyword is not allowed here."), nil)

		}
	case TokenSymbol:
		if t.Val == "[" {
			// Parsing an array literal [expr {, expr}]
			return p.parseArray()
		}
	}

	resolver := &variableResolver{
		locationToken: t,
	}

	if t.Typ != TokenIdentifier {
		// First part of a variable MUST be an identifier

		return nil, p.Error(fmt.Errorf("Expected either a number, string, keyword or identifier."), t)
	}

	resolver.parts = append(resolver.parts, &variablePart{
		typ: varTypeIdent,
		s:   t.Val,
	})
	p.Consume() // we consumed the first identifier of the variable name

variableLoop:
	for p.Remaining() > 0 {
		if p.Match(TokenSymbol, ".") != nil {
			// Next variable part (can be either NUMBER or IDENT)
			t2 := p.Current()
			if t2 != nil {
				switch t2.Typ {
				case TokenIdentifier:
					resolver.parts = append(resolver.parts, &variablePart{
						typ: varTypeIdent,
						s:   t2.Val,
					})
					p.Consume() // consume: IDENT
					continue variableLoop

				case TokenNumber:
					i, err := strconv.Atoi(t2.Val)
					if err != nil {
						return nil, p.Error(err, t2)
					}
					resolver.parts = append(resolver.parts, &variablePart{
						typ: varTypeInt,
						i:   i,
					})
					p.Consume() // consume: NUMBER
					continue variableLoop

				case TokenNil:
					resolver.parts = append(resolver.parts, &variablePart{
						typ:   varTypeNil,
						isNil: true,
					})
					p.Consume() // consume: NIL
					continue variableLoop

				case TokenSymbol:
					if t2.Val != "@" {
						return nil, p.Error(fmt.Errorf("Unexpected symbol %s at the beginning of the identifier.", t2.Val), t2)
					}
					p.Consume() // consume: @

					// Next part must be an IDENT.
					t2 = p.Current()
					if t2 == nil {
						return nil, p.Error(fmt.Errorf("Unexpected EOF, expected either attr IDENTIFIER after @."), p.lastToken)
					} else if t2.Typ != TokenIdentifier {
						return nil, p.Error(fmt.Errorf("This token is not allowed within an identifier name."), t2)
					}

					attrPart := &variablePart{
						typ:            varTypeAttr,
						s:              t2.Val,
						isFunctionCall: true,
					}
					resolver.parts = append(resolver.parts, attrPart)
					p.Consume() // consume: IDENT

					if p.Match(TokenSymbol, "(") != nil {
						// Attribute call with parameters
						// @ AttrName '(' Comma-separated list of expressions ')'
					attrArgsLoop:
						for {
							if p.Remaining() == 0 {
								return nil, p.Error(
									fmt.Errorf("Unexpected EOF, expected function call argument list."),
									p.lastToken)
							}

							if p.Peek(TokenSymbol, ")") == nil {
								// No closing bracket, so we're parsing an expression
								exprArg, err := p.ParseExpression()
								if err != nil {
									return nil, err
								}
								attrPart.callingArgs = append(attrPart.callingArgs, exprArg)

								if p.Match(TokenSymbol, ")") != nil {
									// If there's a closing bracket after an expression, we will stop parsing the arguments
									break attrArgsLoop
								} else {
									// If there's NO closing bracket, there MUST be an comma
									if p.Match(TokenSymbol, ",") == nil {
										return nil,
											p.Error(fmt.Errorf("Missing comma or closing bracket after argument."),
												nil)
									}
								}
							} else {
								// We got a closing bracket, so stop parsing arguments
								p.Consume()
								break attrArgsLoop
							}
						}
					}
					continue variableLoop

				default:
					return nil, p.Error(fmt.Errorf("This token is not allowed within a variable name."), t2)
				}
			} else {
				// EOF
				return nil, p.Error(fmt.Errorf("Unexpected EOF, expected either IDENTIFIER or NUMBER after DOT."),
					p.lastToken)
			}
		} else if p.Match(TokenSymbol, "[") != nil {
			// Variable subscript
			if p.Remaining() == 0 {
				return nil, p.Error(fmt.Errorf("Unexpected EOF, expected subscript subscript."), p.lastToken)
			}

			exprSubscript, err := p.ParseExpression()
			if err != nil {
				return nil, err
			}
			resolver.parts = append(resolver.parts, &variablePart{
				typ:       varTypeSubscript,
				subscript: exprSubscript,
			})
			if p.Match(TokenSymbol, "]") == nil {
				return nil, p.Error(fmt.Errorf("Missing closing bracket after subscript argument."), nil)
			}

		} else if p.Match(TokenSymbol, "(") != nil {
			// Function call
			// FunctionName '(' Comma-separated list of expressions ')'
			part := resolver.parts[len(resolver.parts)-1]
			part.isFunctionCall = true
		argumentLoop:
			for {
				if p.Remaining() == 0 {
					return nil, p.Error(fmt.Errorf("Unexpected EOF, expected function call argument list."), p.lastToken)
				}

				if p.Peek(TokenSymbol, ")") == nil {
					// No closing bracket, so we're parsing an expression
					exprArg, err := p.ParseExpression()
					if err != nil {
						return nil, err
					}
					part.callingArgs = append(part.callingArgs, exprArg)

					if p.Match(TokenSymbol, ")") != nil {
						// If there's a closing bracket after an expression, we will stop parsing the arguments
						break argumentLoop
					} else {
						// If there's NO closing bracket, there MUST be an comma
						if p.Match(TokenSymbol, ",") == nil {
							return nil, p.Error(fmt.Errorf("Missing comma or closing bracket after argument."), nil)
						}
					}
				} else {
					// We got a closing bracket, so stop parsing arguments
					p.Consume()
					break argumentLoop
				}

			}
			// We're done parsing the function call, next variable part
			continue variableLoop
		}

		// No dot, subscript or function call? Then we're done with the variable parsing
		break
	}

	return resolver, nil
}

func (p *Parser) parseVariableOrLiteralWithFilter() (*nodeFilteredVariable, *Error) {
	v := &nodeFilteredVariable{
		locationToken: p.Current(),
	}

	// Parse the variable name
	resolver, err := p.parseVariableOrLiteral()
	if err != nil {
		return nil, err
	}
	v.resolver = resolver

	// Parse all the filters
filterLoop:
	for p.Match(TokenSymbol, "|") != nil {
		// Parse one single filter
		filter, err := p.parseFilter()
		if err != nil {
			return nil, err
		}

		// Check sandbox filter restriction
		if _, isBanned := p.template.set.bannedFilters[filter.name]; isBanned {
			return nil, p.Error(fmt.Errorf("Usage of filter '%s' is not allowed (sandbox restriction active).", filter.name), nil)
		}

		v.filterChain = append(v.filterChain, filter)

		continue filterLoop
	}

	return v, nil
}

func (p *Parser) parseVariableElement() (INode, *Error) {
	node := &nodeVariable{
		locationToken: p.Current(),
	}

	p.Consume() // consume '{{'

	expr, err := p.ParseExpression()
	if err != nil {
		return nil, err
	}
	node.expr = expr

	if p.Match(TokenSymbol, "}}") == nil {
		return nil, p.Error(fmt.Errorf("'}}' expected"), nil)
	}

	return node, nil
}
