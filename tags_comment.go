package pongo2

import "fmt"

type tagCommentNode struct{}

func (node *tagCommentNode) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	return nil
}

func tagCommentParser(doc *Parser, start *Token, arguments *Parser) (INodeTag, *Error) {
	commentNode := &tagCommentNode{}

	// TODO: Process the endtag's arguments (see django 'comment'-tag documentation)
	err := doc.SkipUntilTag("endcomment")
	if err != nil {
		return nil, err
	}

	if arguments.Count() != 0 {
		return nil, arguments.Error(fmt.Errorf("Tag 'comment' does not take any argument."), nil)
	}

	return commentNode, nil
}

func init() {
	MustRegisterTag("comment", tagCommentParser)
}
