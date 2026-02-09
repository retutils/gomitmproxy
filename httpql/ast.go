package httpql

import "fmt"

// AST Nodes mirroring HTTPQL structure

type Query struct {
	Req  *RequestClause
	Resp *ResponseClause
	// Logical operations
	And []*Query
	Or  []*Query
	// TODO: Preset, Row, Source if needed
}

func (q *Query) String() string {
	if q.Req != nil {
		return q.Req.String()
	}
	if q.Resp != nil {
		return q.Resp.String()
	}
	if len(q.And) == 2 {
		return fmt.Sprintf("(%s AND %s)", q.And[0].String(), q.And[1].String())
	}
	if len(q.Or) == 2 {
		return fmt.Sprintf("(%s OR %s)", q.Or[0].String(), q.Or[1].String())
	}
	return ""
}

type RequestClause struct {
	Method *StringExpr
	Host   *StringExpr
	Path   *StringExpr
	Query  *StringExpr
	Body   *StringExpr // Alias: raw
	Port   *IntExpr
	IsTLS  *BoolExpr
	// ... other fields
}

func (r *RequestClause) String() string {
	if r.Method != nil {
		return fmt.Sprintf("req.method.%s", r.Method.String())
	}
	if r.Host != nil {
		return fmt.Sprintf("req.host.%s", r.Host.String())
	}
	if r.Path != nil {
		return fmt.Sprintf("req.path.%s", r.Path.String())
	}
	if r.Query != nil {
		return fmt.Sprintf("req.query.%s", r.Query.String())
	}
	if r.Body != nil {
		return fmt.Sprintf("req.body.%s", r.Body.String())
	}
	if r.Port != nil {
		return fmt.Sprintf("req.port.%s", r.Port.String())
	}
	if r.IsTLS != nil {
		return fmt.Sprintf("req.tls.%s", r.IsTLS.String())
	}
	return ""
}

type ResponseClause struct {
	StatusCode *IntExpr
	Body       *StringExpr // Alias: raw
	Length     *IntExpr
}

func (r *ResponseClause) String() string {
	if r.StatusCode != nil {
		return fmt.Sprintf("resp.code.%s", r.StatusCode.String())
	}
	if r.Body != nil {
		return fmt.Sprintf("resp.body.%s", r.Body.String())
	}
	if r.Length != nil {
		return fmt.Sprintf("resp.len.%s", r.Length.String())
	}
	return ""
}

type StringExpr struct {
	Value    string
	Operator StringOp
}

func (s *StringExpr) String() string {
	return fmt.Sprintf("%s:%q", s.Operator, s.Value)
}

type StringOp string

const (
	OpEq     StringOp = "eq"
	OpNe     StringOp = "ne"
	OpCont   StringOp = "cont"
	OpNCont  StringOp = "ncont"
	OpLike   StringOp = "like"
	OpNLike  StringOp = "nlike"
	OpRegex  StringOp = "regex"
	OpNRegex StringOp = "nregex"
)

type IntExpr struct {
	Value    int
	Operator IntOp
}

func (i *IntExpr) String() string {
	return fmt.Sprintf("%s:%d", i.Operator, i.Value)
}

type IntOp string

const (
	OpIntEq  IntOp = "eq"
	OpIntNe  IntOp = "ne"
	OpIntGt  IntOp = "gt"
	OpIntGte IntOp = "gte"
	OpIntLt  IntOp = "lt"
	OpIntLte IntOp = "lte"
)

type BoolExpr struct {
	Value    bool
	Operator BoolOp
}

func (b *BoolExpr) String() string {
	return fmt.Sprintf("%s:%v", b.Operator, b.Value)
}

type BoolOp string

const (
	OpBoolEq BoolOp = "eq"
	OpBoolNe BoolOp = "ne"
)
