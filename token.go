package orm

type Token int

const (
	ILLEGAL Token = iota
	EOF

	literal_begin
	IDENTIFIER

	SELECT
	FROM
	literal_end

	operator_begin
	COMMA
	operator_end
)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",

	IDENTIFIER: "IDENTIFIER",

	SELECT: "SELECT",
	FROM:   "FROM",

	COMMA: ",",
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
