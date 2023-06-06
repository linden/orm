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

func assemble(element reflect.Type, table string, arguments []any) (string, []any) {
	statement := []Node{
		Node{
			Token: SELECT,
		},
		Node{
			Token: SPACE,
		},
	}

	fields := element.NumField()

	for index := 0; index < fields; index++ {
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

	if len(arguments) > 0 {
		raw, isString := arguments[0].(string)

		if isString == true {
			compiled += " " + raw
			arguments = arguments[1:]
		}
	}

	return compiled, arguments
}

func Scan(connection *pgx.Conn, destination any, table string, arguments ...any) error {
	element := reflect.TypeOf(destination).Elem().Elem()
	value := reflect.ValueOf(destination).Elem()

	statement, arguments := assemble(element, table, arguments)

	rows, err := connection.Query(context.TODO(), statement, arguments...)

	if err != nil {
		return nil
	}

	defer rows.Close()

	for rows.Next() {
		cursor := reflect.New(element).Elem()

		var parameters []any

		for index := 0; index < element.NumField(); index++ {
			parameters = append(parameters, cursor.Field(index).Addr().Interface())
		}

		err = rows.Scan(parameters...)

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

	statement, arguments := assemble(element, table, arguments)

	var parameters []any

	for index := 0; index < element.NumField(); index++ {
		parameters = append(parameters, value.Field(index).Addr().Interface())
	}

	return connection.QueryRow(context.TODO(), statement, arguments...).Scan(parameters...)
}
