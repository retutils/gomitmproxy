package httpql

import (
	"net/url"
	"testing"

	"github.com/retutils/gomitmproxy/proxy"
	uuid "github.com/satori/go.uuid"
)

func TestEvaluator_NilQuery(t *testing.T) {
    var q *Query
    if !q.Eval(nil) {
        t.Error("Expected true for nil query eval")
    }
}

func TestLexer(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input: `req.method.eq:"POST"`,
			expected: []string{
				"req", ".", "method", ".", "eq", ":", "POST", "",
			},
		},
		{
			input: `(req.host.cont:"api" AND resp.code.gte:400)`,
			expected: []string{
				"(", "req", ".", "host", ".", "cont", ":", "api",
				"and",
				"resp", ".", "code", ".", "gte", ":", "400", ")", "",
			},
		},
		{
			input: `req.port.eq:8080 OR req.tls.eq:true`,
			expected: []string{
				"req", ".", "port", ".", "eq", ":", "8080",
				"or",
				"req", ".", "tls", ".", "eq", ":", "true", "",
			},
		},
        {
			input: `NOT req.method.eq:"GET"`,
			expected: []string{
				"not", "req", ".", "method", ".", "eq", ":", "GET", "",
			},
		},
        {
            input: `#`,
            expected: []string{"#", ""},
        },
	}

	for i, tt := range tests {
		lexer := NewLexer(tt.input)
		for j, exp := range tt.expected {
			tok := lexer.NextToken()
			if tok.Literal != exp {
				t.Errorf("Test %d: Expected token %d to be %q, got %q (Type: %d)", i, j, exp, tok.Literal, tok.Type)
			}
		}
	}
}

func TestEvaluator_UnknownOperators(t *testing.T) {
    s := &StringExpr{Value: "v", Operator: "unknown"}
    if s.Eval("v") {
        t.Error("Expected false for unknown operator")
    }

    i := &IntExpr{Value: 1, Operator: "unknown"}
    if i.Eval(1) {
        t.Error("Expected false for unknown int operator")
    }

    b := &BoolExpr{Value: true, Operator: "unknown"}
    if b.Eval(true) {
        t.Error("Expected false for unknown bool operator")
    }
}

func TestParser_ErrorPaths(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"Invalid Namespace", `unknown.field.eq:val`},
        {"Missing Dot", `req field.eq:val`},
        {"Invalid Colon", `req.field.eq val`},
        {"Invalid Int", `req.port.eq:abc`},
        {"Invalid Resp Int", `resp.code.eq:abc`},
        {"Invalid Resp Len Int", `resp.len.eq:abc`},
        {"Unknown Field", `req.unknown.eq:val`},
        {"Unknown Resp Field", `resp.unknown.eq:val`},
        {"Missing Paren", `(req.method.eq:"GET"`},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            l := NewLexer(tt.input)
            p := NewParser(l)
            _, err := p.ParseQuery()
            if err == nil {
                t.Errorf("Expected error for %s", tt.name)
            }
        })
    }
}

func TestParser(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
		check     func(*Query) bool
	}{
		{
			name:  "Simple Request",
			input: `req.method.eq:"GET"`,
			check: func(q *Query) bool {
				return q.Req != nil && q.Req.Method != nil && q.Req.Method.Value == "GET" && q.Req.Method.Operator == OpEq
			},
		},
		{
			name:  "Simple Response",
			input: `resp.code.ne:200`,
			check: func(q *Query) bool {
				return q.Resp != nil && q.Resp.StatusCode != nil && q.Resp.StatusCode.Value == 200 && q.Resp.StatusCode.Operator == OpIntNe
			},
		},
        {
			name:  "Response Len",
			input: `resp.len.gt:100`,
			check: func(q *Query) bool {
				return q.Resp != nil && q.Resp.Length != nil && q.Resp.Length.Value == 100
			},
		},
        {
			name:  "Response Body",
			input: `resp.body.cont:"error"`,
			check: func(q *Query) bool {
				return q.Resp != nil && q.Resp.Body != nil && q.Resp.Body.Value == "error"
			},
		},
		{
			name:  "AND Logic",
			input: `req.method.eq:"POST" AND req.path.like:"/api/*"`,
			check: func(q *Query) bool {
				return len(q.And) == 2 && q.And[0].Req.Method.Value == "POST" && q.And[1].Req.Path.Operator == OpLike
			},
		},
		{
			name:  "OR Logic",
			input: `resp.code.eq:404 OR resp.code.eq:500`,
			check: func(q *Query) bool {
				return len(q.Or) == 2
			},
		},
		{
			name:  "Nested Logic (Parens)",
			input: `req.method.eq:"POST" AND (resp.code.eq:200 OR resp.code.eq:201)`,
			check: func(q *Query) bool {
				return len(q.And) == 2 && len(q.And[1].Or) == 2
			},
		},
		{
			name:      "Invalid Field",
			input:     `req.unknown.eq:"foo"`,
			shouldErr: true,
		},
		{
			name:      "Invalid Operator Syntax",
			input:     `req.method.eq`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.input)
			p := NewParser(l)
			q, err := p.ParseQuery()

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if q != nil && tt.check != nil {
					if !tt.check(q) {
						t.Errorf("Query check failed for input: %s", tt.input)
					}
				}
			}
		})
	}
}

func TestEvaluator(t *testing.T) {
	flow := &proxy.Flow{
		Id: uuid.NewV4(),
		Request: &proxy.Request{
			Method: "POST",
			URL: &url.URL{
				Scheme: "https",
				Host:   "api.example.com",
				Path:   "/v1/users",
			},
			Proto: "HTTP/1.1",
			Body:  []byte(`{"user_id": 123, "name": "test"}`),
		},
		Response: &proxy.Response{
			StatusCode: 201,
			Body:       []byte(`{"status": "created", "id": 123}`),
		},
	}

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		// String Exact Matches
		{"Method Eq", `req.method.eq:"POST"`, true},
		{"Method Ne", `req.method.ne:"GET"`, true},
		{"Method Eq False", `req.method.eq:"GET"`, false},
		
		// String Contains
		{"Host Cont", `req.host.cont:"example"`, true},
		{"Host Cont False", `req.host.cont:"google"`, false},
		{"Host NCont", `req.host.ncont:"google"`, true},

		// String Like (Wildcard)
		{"Path Like", `req.path.like:"/v1/*"`, true},
		{"Path Like Mid", `req.path.like:"*/users"`, true},
		{"Path Like False", `req.path.like:"/v2/*"`, false},
		{"Path NLike", `req.path.nlike:"/v2/*"`, true},

		// String Regex
		{"Host Regex", `req.host.regex:"^api\..+\.com$"`, true},
		{"Host Regex False", `req.host.regex:"^www\."`, false},
		{"Host NRegex", `req.host.nregex:"^www\."`, true},

		// Int Comparisons
		{"Status Eq", `resp.code.eq:201`, true},
		{"Status Gt Link", `resp.code.gt:200`, true},
		{"Status Lt", `resp.code.lt:300`, true},
		{"Status Gte", `resp.code.gte:201`, true},
		{"Status Lte", `resp.code.lte:201`, true},
		{"Status Eq False", `resp.code.eq:200`, false},

		// Logical Ops
		{"AND True", `req.method.eq:"POST" AND resp.code.eq:201`, true},
		{"AND False", `req.method.eq:"POST" AND resp.code.eq:200`, false},
		{"OR True 1", `req.method.eq:"GET" OR resp.code.eq:201`, true}, // 2nd true
		{"OR True 2", `req.method.eq:"POST" OR resp.code.eq:500`, true}, // 1st true
		{"OR False", `req.method.eq:"GET" OR resp.code.eq:500`, false},

		// Nested
		{"Nested True", `req.host.cont:"example" AND (resp.code.eq:200 OR resp.code.eq:201)`, true},
		{"Nested False", `req.host.cont:"example" AND (resp.code.eq:400 OR resp.code.eq:404)`, false},

		// Body Search
		{"Req Body Cont", `req.body.cont:"user_id"`, true},
		{"Req Body Eq False", `req.body.eq:"full_body"`, false},
		{"Resp Body Like", `resp.body.like:"*created*"`, true},

		// Port and TLS
		{"Port Eq", `req.port.eq:443`, true},
		{"TLS Eq", `req.tls.eq:true`, true},
		{"TLS Ne", `req.tls.ne:false`, true},

		// Response Length
		{"Resp Len Gt", `resp.len.gt:10`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.query)
			p := NewParser(l)
			q, err := p.ParseQuery()
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			
			result := q.Eval(flow)
			if result != tt.expected {
				t.Errorf("Eval(%s) = %v; want %v", tt.query, result, tt.expected)
			}
		})
	}
}

func TestEvaluator_NilResponse(t *testing.T) {
	flow := &proxy.Flow{
		Request: &proxy.Request{
			Method: "GET",
			URL: &url.URL{Host: "example.com"},
		},
		Response: nil,
	}

	query := `resp.code.eq:200`
	l := NewLexer(query)
	p := NewParser(l)
	q, _ := p.ParseQuery()
	
	if q.Eval(flow) {
		t.Errorf("Eval should return false for nil response")
	}

	query2 := `req.method.eq:"GET"`
	l2 := NewLexer(query2)
	p2 := NewParser(l2)
	q2, _ := p2.ParseQuery()

	if !q2.Eval(flow) {
		t.Errorf("Eval should return true for request check even if response is nil")
	}
}

func TestParser_Errors(t *testing.T) {
	tests := []struct {
		query string
	}{
		{"(req.method.eq:\"GET\""}, // Missing )
		{"invalid.method.eq:\"GET\""}, // Invalid namespace
		{"req.invalid.eq:\"GET\""}, // Invalid field
		{"req.method.eq"}, // Missing :val
	}
	for _, tt := range tests {
		l := NewLexer(tt.query)
		p := NewParser(l)
		_, err := p.ParseQuery()
		if err == nil {
			t.Errorf("Expected error for query: %s", tt.query)
		}
	}
}
