package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/linden/orm"
)

// CREATE TABLE people (id SERIAL PRIMARY KEY, name TEXT, age INT);
// CREATE TABLE friends (id SERIAL, person_a_id INT REFERENCES people(id), person_b_id INT REFERENCES people(id));
// INSERT INTO people (id, name, age) VALUES (1, 'bob', 24);
// INSERT INTO people (id, name, age) VALUES (2, 'jim', 28);
// INSERT INTO friends (person_a_id, person_b_id) VALUES (1, 2);

type Person struct {
	ID   int    `orm:"id"`
	Name string `orm:"name"`
}

type Friend struct {
	ID      int    `orm:"id"`
	PersonA Person `orm_foreign:"people,id,person_a_id"`
	PersonB Person `orm_foreign:"people,id,person_b_id"`
}

func main() {
	connection, err := pgx.Connect(context.TODO(), "postgres://postgres:postgres@localhost:5432/postgres")

	if err != nil {
		panic(err)
	}

	defer connection.Close(context.TODO())

	var person Person

	err = orm.ScanRow(context.TODO(), connection, &person, "people", "WHERE age = $1", 24)

	if err != nil {
		panic(err)
	}

	fmt.Printf("person: %+v\n", person)

	var people []Person

	err = orm.Scan(context.TODO(), connection, &people, "people")

	if err != nil {
		panic(err)
	}

	fmt.Printf("people: %+v\n", people)

	var friend Friend

	err = orm.ScanRow(context.TODO(), connection, &friend, "friends")

	if err != nil {
		panic(err)
	}

	fmt.Printf("friend: %+v\n", friend)
}
