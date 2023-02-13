package lexer

import (
	"testing"

	"go.ddosify.com/ddosify/core/scenario/scripting/assertion/token"
)

func TestNextToken(t *testing.T) {
	tests := []struct {
		input    string
		expected []struct {
			expectedType    token.TokenType
			expectedLiteral string
		}
	}{
		{
			input: "range(headers_content_length, 100, 300)",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "range"},
				{token.LPAREN, "("},
				{token.IDENT, "headers_content_length"},
				{token.COMMA, ","},
				{token.INT, "100"},
				{token.COMMA, ","},
				{token.INT, "300"},
				{token.RPAREN, ")"},
				{token.EOF, ""},
			},
		},
		{
			input: "in(status_code, [200, 201])",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "in"},
				{token.LPAREN, "("},
				{token.IDENT, "status_code"},
				{token.COMMA, ","},
				{token.LBRACKET, "["},
				{token.INT, "200"},
				{token.COMMA, ","},
				{token.INT, "201"},
				{token.RBRACKET, "]"},
				{token.RPAREN, ")"},
				{token.EOF, ""},
			},
		},
		{
			input: "not(in(status_code, [200, 201]))",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "not"},
				{token.LPAREN, "("},
				{token.IDENT, "in"},
				{token.LPAREN, "("},
				{token.IDENT, "status_code"},
				{token.COMMA, ","},
				{token.LBRACKET, "["},
				{token.INT, "200"},
				{token.COMMA, ","},
				{token.INT, "201"},
				{token.RBRACKET, "]"},
				{token.RPAREN, ")"},
				{token.RPAREN, ")"},
				{token.EOF, ""},
			},
		},
		{
			input: "in(headers_Content_Type, [\"application/json\", \"application/xml\"])",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "in"},
				{token.LPAREN, "("},
				{token.IDENT, "headers_Content_Type"},
				{token.COMMA, ","},
				{token.LBRACKET, "["},
				{token.STRING, "application/json"},
				{token.COMMA, ","},
				{token.STRING, "application/xml"},
				{token.RBRACKET, "]"},
				{token.RPAREN, ")"},
				{token.EOF, ""},
			},
		},
		{
			input: "equals(json_path(employees.percentage), 32.3)",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "equals"},
				{token.LPAREN, "("},
				{token.IDENT, "json_path"},
				{token.LPAREN, "("},
				{token.IDENT, "employees.percentage"},
				{token.RPAREN, ")"},
				{token.COMMA, ","},
				{token.FLOAT, "32.3"},
				{token.RPAREN, ")"},
				{token.EOF, ""},
			},
		},
		{
			input: "2+5-3*10/2<>",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.INT, "2"},
				{token.PLUS, "+"},
				{token.INT, "5"},
				{token.MINUS, "-"},
				{token.INT, "3"},
				{token.ASTERISK, "*"},
				{token.INT, "10"},
				{token.SLASH, "/"},
				{token.INT, "2"},
				{token.LT, "<"},
				{token.GT, ">"},
				{token.EOF, ""},
			},
		},
		{
			input: "response_size == 234",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "response_size"},
				{token.EQ, "=="},
				{token.INT, "234"},
				{token.EOF, ""},
			},
		},
		{
			input: "response_size != 234",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "response_size"},
				{token.NOT_EQ, "!="},
				{token.INT, "234"},
				{token.EOF, ""},
			},
		},
		{
			input: "!has(headers.referrer)",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.BANG, "!"},
				{token.IDENT, "has"},
				{token.LPAREN, "("},
				{token.IDENT, "headers.referrer"},
				{token.RPAREN, ")"},
				{token.EOF, ""},
			},
		},
		{
			input: "a = 5",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "a"},
				{token.ILLEGAL, "="},
				{token.INT, "5"},
				{token.EOF, ""},
			},
		},
		{
			input: "60.1 $ 60.1",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.FLOAT, "60.1"},
				{token.ILLEGAL, "$"},
				{token.FLOAT, "60.1"},
				{token.EOF, ""},
			},
		},
		{
			input: "%",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.ILLEGAL, "%"},
				{token.EOF, ""},
			},
		},
		{
			input: "a =",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "a"},
				{token.ILLEGAL, "="},
				{token.EOF, ""},
			},
		},
		{
			input: "not(true) && not(false)",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "not"},
				{token.LPAREN, "("},
				{token.TRUE, "true"},
				{token.RPAREN, ")"},
				{token.AND, "&&"},
				{token.IDENT, "not"},
				{token.LPAREN, "("},
				{token.FALSE, "false"},
				{token.RPAREN, ")"},
				{token.EOF, ""},
			},
		},
		{
			input: "equals(status_code,200) || equals(status_code,201)",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "equals"},
				{token.LPAREN, "("},
				{token.IDENT, "status_code"},
				{token.COMMA, ","},
				{token.INT, "200"},
				{token.RPAREN, ")"},
				{token.OR, "||"},
				{token.IDENT, "equals"},
				{token.LPAREN, "("},
				{token.IDENT, "status_code"},
				{token.COMMA, ","},
				{token.INT, "201"},
				{token.RPAREN, ")"},
				{token.EOF, ""},
			},
		},
		{
			input: "equals(json_path(),null)",
			expected: []struct {
				expectedType    token.TokenType
				expectedLiteral string
			}{
				{token.IDENT, "equals"},
				{token.LPAREN, "("},
				{token.IDENT, "json_path"},
				{token.LPAREN, "("},
				{token.RPAREN, ")"},
				{token.COMMA, ","},
				{token.NULL, "null"},
				{token.RPAREN, ")"},
				{token.EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		l := New(tt.input)

		var tok token.Token
		i := 0
		for tok = l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
			if tok.Type != tt.expected[i].expectedType {
				t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
					i, tt.expected[i].expectedType, tok.Type)
			}

			if tok.Literal != tt.expected[i].expectedLiteral {
				t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
					i, tt.expected[i].expectedLiteral, tok.Literal)
			}
			i++
		}

	}
}
