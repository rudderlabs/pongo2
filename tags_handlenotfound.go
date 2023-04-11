package pongo2

import (
	"bytes"
)

type tagHandleNode struct {
	position    *Token
	bodyWrapper *NodeWrapper
}

func (node *tagHandleNode) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	temp := bytes.NewBuffer(make([]byte, 0, 1024)) // 1 KiB size
	ctx.HandleError = true

	err := node.bodyWrapper.Execute(ctx, temp)
	if err != nil {
		return err
	}
	templateSet := ctx.template.set
	currentTemplate, err2 := templateSet.FromBytes(temp.Bytes())
	if err2 != nil {
		return err2.(*Error)
	}
	currentTemplate.root.Execute(ctx, writer)
	return nil
}

func tagHandleParser(doc *Parser, start *Token, arguments *Parser) (INodeTag, *Error) {
	execNode := &tagHandleNode{
		position: start,
	}

	wrapper, _, err := doc.WrapUntilTag("endhandlenotfound")
	if err != nil {
		return nil, err
	}
	execNode.bodyWrapper = wrapper

	return execNode, nil
}

func init() {
	RegisterTag("handlenotfound", tagHandleParser)
}