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

	for index, node := range statement {
		if node.Literal == "" {
			buffer.WriteString(node.Token.String())
		} else {
			buffer.WriteString(node.Literal)
		}

		if index+1 != len(statement) {
			buffer.WriteByte(' ')
		}
	}

	return buffer.String()
}
