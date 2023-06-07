package orm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
)

type Foreign struct {
	Fields []Selection

	// JOIN table
	Table string

	// ON left = right
	Left  string
	Right string
}

type Selection struct {
	Name    string
	Foreign Foreign
}

func newSelection(raw reflect.StructField) (Selection, error) {
	foreign := raw.Tag.Get("orm_foreign")

	if foreign == "" {
		name := raw.Name

		tag := raw.Tag.Get("orm")

		if tag != "" {
			name = tag
		}

		return Selection{
			Name: name,
		}, nil
	}

	var fields []Selection

	for index := 0; index < raw.Type.NumField(); index++ {
		field, err := newSelection(raw.Type.Field(index))

		if err != nil {
			return Selection{}, err
		}

		fields = append(fields, field)
	}

	split := strings.Split(foreign, ",")

	if len(split) != 3 {
		return Selection{}, fmt.Errorf("expected 3 fields (table, left, right) got %d", len(split))
	}

	return Selection{
		Foreign: Foreign{
			Fields: fields,

			Table: split[0],

			Left:  split[1],
			Right: split[2],
		},
	}, nil
}

func assembleForeignStatement(foreign Foreign) []Node {
	var statement []Node

	for index, field := range foreign.Fields {
		statement = append(statement, []Node{
			Node{
				Token: QUOTE,
			},
			Node{
				Token:   IDENTIFIER,
				Literal: foreign.Right + "_reference",
			},
			Node{
				Token: QUOTE,
			},
			Node{
				Token: PERIOD,
			},
			Node{
				Token: QUOTE,
			},
			Node{
				Token:   IDENTIFIER,
				Literal: field.Name,
			},
			Node{
				Token: QUOTE,
			},
		}...)

		if index+1 != len(foreign.Fields) {
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

	return statement
}

func assembleForeignJoins(table string, foreign Foreign) []Node {
	alias := foreign.Right + "_reference"

	return []Node{
		Node{
			Token: INNER_JOIN,
		},
		Node{
			Token: SPACE,
		},
		Node{
			Token:   IDENTIFIER,
			Literal: foreign.Table,
		},
		Node{
			Token: SPACE,
		},
		Node{
			Token:   IDENTIFIER,
			Literal: alias,
		},
		Node{
			Token: SPACE,
		},
		Node{
			Token: ON,
		},
		Node{
			Token: SPACE,
		},
		Node{
			Token:   IDENTIFIER,
			Literal: alias,
		},
		Node{
			Token: PERIOD,
		},
		Node{
			Token:   IDENTIFIER,
			Literal: foreign.Left,
		},
		Node{
			Token: SPACE,
		},
		Node{
			Token: EQUAL,
		},
		Node{
			Token: SPACE,
		},
		Node{
			Token:   IDENTIFIER,
			Literal: table,
		},
		Node{
			Token: PERIOD,
		},
		Node{
			Token:   IDENTIFIER,
			Literal: foreign.Right,
		},
		Node{
			Token: SPACE,
		},
	}
}

func assembleParameters(element reflect.Type, value reflect.Value) []any {
	var parameters []any

	for index := 0; index < element.NumField(); index++ {
		field := value.Field(index)
		foreign := element.Field(index).Tag.Get("orm_foreign")

		if foreign != "" {
			for nested := 0; nested < field.NumField(); nested++ {
				parameters = append(parameters, field.Field(nested).Addr().Interface())
			}
		} else {
			parameters = append(parameters, field.Addr().Interface())
		}
	}

	return parameters
}

func assemble(element reflect.Type, table string, arguments []any) (string, []any, error) {
	statement := []Node{
		Node{
			Token: SELECT,
		},
		Node{
			Token: SPACE,
		},
	}

	fields := element.NumField()

	var selections []Selection
	var joins []Node

	for index := 0; index < fields; index++ {
		field := element.Field(index)
		selection, err := newSelection(field)

		if err != nil {
			return "", []any{}, err
		}

		selections = append(selections, selection)

		if selection.Name != "" {
			statement = append(statement, []Node{
				Node{
					Token: QUOTE,
				},
				Node{
					Token:   IDENTIFIER,
					Literal: table,
				},
				Node{
					Token: QUOTE,
				},
				Node{
					Token: PERIOD,
				},
				Node{
					Token: QUOTE,
				},
				Node{
					Token:   IDENTIFIER,
					Literal: selection.Name,
				},
				Node{
					Token: QUOTE,
				},
			}...)
		} else {
			statement = append(statement, assembleForeignStatement(selection.Foreign)...)
			joins = append(joins, assembleForeignJoins(table, selection.Foreign)...)
		}

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
		Node{
			Token: SPACE,
		},
	}...)

	statement = append(statement, joins...)

	compiled := Compile(statement)

	if len(arguments) > 0 {
		raw, isString := arguments[0].(string)

		if isString == true {
			compiled += " " + raw
			arguments = arguments[1:]
		}
	}

	return compiled, arguments, nil
}

func Scan(connection *pgx.Conn, destination any, table string, arguments ...any) error {
	element := reflect.TypeOf(destination).Elem().Elem()
	value := reflect.ValueOf(destination).Elem()

	statement, arguments, err := assemble(element, table, arguments)

	if err != nil {
		return err
	}

	rows, err := connection.Query(context.TODO(), statement, arguments...)

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		cursor := reflect.New(element).Elem()

		err = rows.Scan(assembleParameters(element, cursor)...)

		if err != nil {
			return err
		}

		value.Set(reflect.Append(value, cursor))
	}

	return nil
}

func ScanRow(connection *pgx.Conn, destination any, table string, arguments ...any) error {
	element := reflect.TypeOf(destination).Elem()
	value := reflect.ValueOf(destination).Elem()

	statement, arguments, err := assemble(element, table, arguments)

	if err != nil {
		return err
	}

	return connection.QueryRow(context.TODO(), statement, arguments...).Scan(assembleParameters(element, value)...)
}
