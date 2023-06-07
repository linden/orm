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

type RowQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
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

func assembleParameters(element reflect.Type, value reflect.Value) ([]any, error) {
	var parameters []any

	for index := 0; index < element.NumField(); index++ {
		valueField := value.Field(index)
		elementField := element.Field(index)

		foreign := elementField.Tag.Get("orm_foreign")

		if foreign != "" {
			kind := elementField.Type.Kind()

			if kind != reflect.Struct {
				return []any{}, fmt.Errorf("foreign fields must be structs: field %s is a %s", elementField.Name, kind)
			}

			for nested := 0; nested < valueField.NumField(); nested++ {
				parameters = append(parameters, valueField.Field(nested).Addr().Interface())
			}
		} else {
			parameters = append(parameters, valueField.Addr().Interface())
		}
	}

	return parameters, nil
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

func Scan(session context.Context, querier Querier, destination any, table string, arguments ...any) error {
	pointer := reflect.TypeOf(destination).Kind()

	if pointer != reflect.Pointer {
		return fmt.Errorf("destination must be a pointer to an array of structs: got a %s", pointer)
	}

	array := reflect.TypeOf(destination).Elem().Kind()

	if array != reflect.Slice {
		return fmt.Errorf("destination must be a pointer to an array of structs: got a pointer to a %s", array)
	}

	element := reflect.TypeOf(destination).Elem().Elem()

	if element.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to an array of structs: got a pointer to an array of %ss", element.Kind())
	}

	value := reflect.ValueOf(destination).Elem()

	statement, arguments, err := assemble(element, table, arguments)

	if err != nil {
		return err
	}

	rows, err := querier.Query(session, statement, arguments...)

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		cursor := reflect.New(element).Elem()

		parameters, err := assembleParameters(element, cursor)

		if err != nil {
			return err
		}

		err = rows.Scan(parameters...)

		if err != nil {
			return err
		}

		value.Set(reflect.Append(value, cursor))
	}

	return nil
}

func ScanRow(session context.Context, querier RowQuerier, destination any, table string, arguments ...any) error {
	kind := reflect.TypeOf(destination).Kind()

	if kind != reflect.Pointer {
		return fmt.Errorf("destination must be a pointer to a struct: got kind %s", kind)
	}

	element := reflect.TypeOf(destination).Elem()
	value := reflect.ValueOf(destination).Elem()

	if element.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to a struct: got a pointer to kind %s", element.Kind())
	}

	statement, arguments, err := assemble(element, table, arguments)

	if err != nil {
		return err
	}

	parameters, err := assembleParameters(element, value)

	if err != nil {
		return err
	}

	return querier.QueryRow(session, statement, arguments...).Scan(parameters...)
}
