package orm

import (
	"context"
	"reflect"

	"github.com/jackc/pgx/v5"
)

type Brand struct {
	ID     string
	Domain string `orm:"domain"`
}

func ScanRow(connection *pgx.Conn, destination any, table string, extra ...any) error {
	element := reflect.TypeOf(destination).Elem()
	value := reflect.ValueOf(destination).Elem()

	statement := []Node{
		Node{
			Token: SELECT,
		},
		Node{
			Token: SPACE,
		},
	}

	fields := element.NumField()

	var arguments []any

	for index := 0; index < fields; index++ {
		arguments = append(arguments, value.Field(index).Addr().Interface())

		field := element.Field(index)

		name := field.Name
		tag := field.Tag.Get("orm")

		if tag != "" {
			name = tag
		}

		statement = append(statement, []Node{
			Node{
				Token: QUOTE,
			},
			Node{
				Token:   IDENTIFIER,
				Literal: name,
			},
			Node{
				Token: QUOTE,
			},
		}...)

		if index+1 != fields {
			statement = append(statement, []Node{
				Node{
					Token: COMMA,
				},
				Node{
					Token: SPACE,
				},
			}...)
		}
	}

	statement = append(statement, []Node{
		Node{
			Token: SPACE,
		},
		Node{
			Token: FROM,
		},
		Node{
			Token: SPACE,
		},
		Node{
			Token:   IDENTIFIER,
			Literal: table,
		},
	}...)

	compiled := Compile(statement)

	if len(extra) > 0 {
		compiled += " " + extra[0].(string)
		extra = extra[1:]
	}

	return connection.QueryRow(context.TODO(), compiled, extra...).Scan(arguments...)
}
