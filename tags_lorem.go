package pongo2

import (
	"fmt"
	"math/rand"
	"strings"
)

const maxLoremCount = 100000

var (
	tagLoremParagraphs = strings.Split(tagLoremText, "\n")
	tagLoremWords      = strings.Fields(tagLoremText)
)

type tagLoremNode struct {
	position *Token
	count    int    // number of paragraphs
	method   string // w = words, p = HTML paragraphs, b = plain-text (default is b)
	random   bool   // does not use the default paragraph "Lorem ipsum dolor sit amet, ..."
}

func (node *tagLoremNode) Execute(ctx *ExecutionContext, writer TemplateWriter) *Error {
	if node.count > maxLoremCount {
		return ctx.Error(fmt.Errorf("max count for lorem is %d", maxLoremCount), node.position)
	}

	switch node.method {
	case "b":
		if node.random {
			for i := 0; i < node.count; i++ {
				if i > 0 {
					if _, err := writer.WriteString("\n"); err != nil {
						return ctx.Error(err, node.position)
					}
				}
				par := tagLoremParagraphs[rand.Intn(len(tagLoremParagraphs))]
				if _, err := writer.WriteString(par); err != nil {
					return ctx.Error(err, node.position)
				}
			}
		} else {
			for i := 0; i < node.count; i++ {
				if i > 0 {
					if _, err := writer.WriteString("\n"); err != nil {
						return ctx.Error(err, node.position)
					}
				}
				par := tagLoremParagraphs[i%len(tagLoremParagraphs)]
				if _, err := writer.WriteString(par); err != nil {
					return ctx.Error(err, node.position)
				}
			}
		}
	case "w":
		if node.random {
			for i := 0; i < node.count; i++ {
				if i > 0 {
					if _, err := writer.WriteString(" "); err != nil {
						return ctx.Error(err, node.position)
					}
				}
				word := tagLoremWords[rand.Intn(len(tagLoremWords))]
				if _, err := writer.WriteString(word); err != nil {
					return ctx.Error(err, node.position)
				}
			}
		} else {
			for i := 0; i < node.count; i++ {
				if i > 0 {
					if _, err := writer.WriteString(" "); err != nil {
						return ctx.Error(err, node.position)
					}
				}
				word := tagLoremWords[i%len(tagLoremWords)]
				if _, err := writer.WriteString(word); err != nil {
					return ctx.Error(err, node.position)
				}
			}
		}
	case "p":
		if node.random {
			for i := 0; i < node.count; i++ {
				if i > 0 {
					if _, err := writer.WriteString("\n"); err != nil {
						return ctx.Error(err, node.position)
					}
				}
				if _, err := writer.WriteString("<p>"); err != nil {
					return ctx.Error(err, node.position)
				}
				par := tagLoremParagraphs[rand.Intn(len(tagLoremParagraphs))]
				if _, err := writer.WriteString(par); err != nil {
					return ctx.Error(err, node.position)
				}
				if _, err := writer.WriteString("</p>"); err != nil {
					return ctx.Error(err, node.position)
				}

			}
		} else {
			for i := 0; i < node.count; i++ {
				if i > 0 {
					if _, err := writer.WriteString("\n"); err != nil {
						return ctx.Error(err, node.position)
					}
				}
				if _, err := writer.WriteString("<p>"); err != nil {
					return ctx.Error(err, node.position)
				}
				par := tagLoremParagraphs[i%len(tagLoremParagraphs)]
				if _, err := writer.WriteString(par); err != nil {
					return ctx.Error(err, node.position)
				}
				if _, err := writer.WriteString("</p>"); err != nil {
					return ctx.Error(err, node.position)
				}

			}
		}
	default:
		return ctx.OrigError(fmt.Errorf("unsupported method: %s", node.method), nil)
	}

	return nil
}

func tagLoremParser(doc *Parser, start *Token, arguments *Parser) (INodeTag, *Error) {
	loremNode := &tagLoremNode{
		position: start,
		count:    1,
		method:   "b",
	}

	if countToken := arguments.MatchType(TokenNumber); countToken != nil {
		loremNode.count = AsValue(countToken.Val).Integer()
	}

	if methodToken := arguments.MatchType(TokenIdentifier); methodToken != nil {
		if methodToken.Val != "w" && methodToken.Val != "p" && methodToken.Val != "b" {
			return nil, arguments.Error(fmt.Errorf("lorem-method must be either 'w', 'p' or 'b'."), nil)
		}

		loremNode.method = methodToken.Val
	}

	if arguments.MatchOne(TokenIdentifier, "random") != nil {
		loremNode.random = true
	}

	if arguments.Remaining() > 0 {
		return nil, arguments.Error(fmt.Errorf("Malformed lorem-tag arguments."), nil)
	}

	return loremNode, nil
}

func init() {
	MustRegisterTag("lorem", tagLoremParser)
}

const tagLoremText = `Lorem ipsum dolor sit amet, consectetur adipisici elit, sed eiusmod tempor incidunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquid ex ea commodi consequat. Quis aute iure reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint obcaecat cupiditat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.
Duis autem vel eum iriure dolor in hendrerit in vulputate velit esse molestie consequat, vel illum dolore eu feugiat nulla facilisis at vero eros et accumsan et iusto odio dignissim qui blandit praesent luptatum zzril delenit augue duis dolore te feugait nulla facilisi. Lorem ipsum dolor sit amet, consectetuer adipiscing elit, sed diam nonummy nibh euismod tincidunt ut laoreet dolore magna aliquam erat volutpat.
Ut wisi enim ad minim veniam, quis nostrud exerci tation ullamcorper suscipit lobortis nisl ut aliquip ex ea commodo consequat. Duis autem vel eum iriure dolor in hendrerit in vulputate velit esse molestie consequat, vel illum dolore eu feugiat nulla facilisis at vero eros et accumsan et iusto odio dignissim qui blandit praesent luptatum zzril delenit augue duis dolore te feugait nulla facilisi.
Nam liber tempor cum soluta nobis eleifend option congue nihil imperdiet doming id quod mazim placerat facer possim assum. Lorem ipsum dolor sit amet, consectetuer adipiscing elit, sed diam nonummy nibh euismod tincidunt ut laoreet dolore magna aliquam erat volutpat. Ut wisi enim ad minim veniam, quis nostrud exerci tation ullamcorper suscipit lobortis nisl ut aliquip ex ea commodo consequat.
Duis autem vel eum iriure dolor in hendrerit in vulputate velit esse molestie consequat, vel illum dolore eu feugiat nulla facilisis.
At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, At accusam aliquyam diam diam dolore dolores duo eirmod eos erat, et nonumy sed tempor et et invidunt justo labore Stet clita ea et gubergren, kasd magna no rebum. sanctus sea sed takimata ut vero voluptua. est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat.
Consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet. Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores et ea rebum. Stet clita kasd gubergren, no sea takimata sanctus est Lorem ipsum dolor sit amet.`
