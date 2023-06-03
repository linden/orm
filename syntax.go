package orm

import (
	"bytes"
)

type Node struct {
	Token   Token
	Literal string
}

func Compile(statement []Node) string {
	buffer := new(bytes.Buffer)

	for _, node := range statement {
		if node.Literal == "" {
			buffer.WriteString(node.Token.String())
		} else {
			buffer.WriteString(node.Literal)
		}
	}

	return buffer.String()
}
