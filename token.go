package orm

type Token int

const (
	ILLEGAL Token = iota
	EOF

	literal_begin
	IDENTIFIER

	INNER_JOIN
	SELECT
	FROM
	ON
	literal_end

	symbol_begin
	COMMA
	SPACE
	QUOTE
	EQUAL
	PERIOD
	symbol_end
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",

	IDENTIFIER: "IDENTIFIER",

	INNER_JOIN: "INNER JOIN",
	SELECT:     "SELECT",
	FROM:       "FROM",
	ON:         "ON",

	COMMA:  ",",
	SPACE:  " ",
	QUOTE:  "\"",
	EQUAL:  "=",
	PERIOD: ".",
}

func (token Token) String() string {
	if len(tokens) <= int(token) {
		return tokens[ILLEGAL]
	}

	if tokens[token] == "" {
		return tokens[ILLEGAL]
	}

	return tokens[token]
}
