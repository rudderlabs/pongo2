package pongo2

import (
	"strings"
)

type nodeHTML struct {
	token     *Token
	trimLeft  bool
	trimRight bool
}

func (n *nodeHTML) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	res := n.token.Val
	if n.trimLeft {
		res = strings.TrimLeft(res, tokenSpaceChars)
	}
	if n.trimRight {
		res = strings.TrimRight(res, tokenSpaceChars)
	}
	_, err := writer.WriteString(res)
	if err != nil {
		return ctx.Error(err, n.token)
	}
	return nil
}
