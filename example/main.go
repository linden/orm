package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/linden/orm"
)

// CREATE TABLE people (ID SERIAL, name TEXT, age INT);
// INSERT INTO people (name, age) VALUES ('bob', 10);

type Person struct {
	ID   int    `orm:"id"`
	Name string `orm:"name"`
}

func main() {
	connection, err := pgx.Connect(context.TODO(), "postgres://postgres:postgres@localhost:5432/postgres")

	if err != nil {
		panic(err)
	}

	defer connection.Close(context.TODO())

	var person Person

	err = orm.ScanRow(connection, &person, "people", "WHERE age = $1", 10)

	if err != nil {
		panic(err)
	}

	fmt.Printf("person: %+v\n", person)
}
