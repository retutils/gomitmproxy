package httpql

import (
	"strings"
	"testing"
)

func TestAST_String(t *testing.T) {
	q := &Query{
		Req: &RequestClause{
			Method: &StringExpr{Value: "GET", Operator: OpEq},
		},
	}
	if s := q.String(); s != `req.method.eq:"GET"` {
		t.Errorf("Expected req.method.eq:\"GET\", got %s", s)
	}

	q2 := &Query{
		Resp: &ResponseClause{
			StatusCode: &IntExpr{Value: 200, Operator: OpIntEq},
		},
	}
	if s := q2.String(); s != "resp.code.eq:200" {
		t.Errorf("Expected resp.code.eq:200, got %s", s)
	}

	q3 := &Query{
		And: []*Query{q, q2},
	}
	if s := q3.String(); s != `(req.method.eq:"GET" AND resp.code.eq:200)` {
		t.Errorf("Unexpected AND string: %s", s)
	}

	q4 := &Query{
		Or: []*Query{q, q2},
	}
	if s := q4.String(); s != `(req.method.eq:"GET" OR resp.code.eq:200)` {
		t.Errorf("Unexpected OR string: %s", s)
	}
	
	// Test other Request fields
	req := &RequestClause{
		Host: &StringExpr{Value: "h", Operator: OpEq},
	}
	if !strings.Contains(req.String(), "req.host.eq") { t.Error("host fail") }
	
	req = &RequestClause{
		Path: &StringExpr{Value: "p", Operator: OpEq},
	}
	if !strings.Contains(req.String(), "req.path.eq") { t.Error("path fail") }

	req = &RequestClause{
		Query: &StringExpr{Value: "q", Operator: OpEq},
	}
	if !strings.Contains(req.String(), "req.query.eq") { t.Error("query fail") }

	req = &RequestClause{
		Body: &StringExpr{Value: "b", Operator: OpEq},
	}
	if !strings.Contains(req.String(), "req.body.eq") { t.Error("body fail") }

	req = &RequestClause{
		Port: &IntExpr{Value: 80, Operator: OpIntEq},
	}
	if !strings.Contains(req.String(), "req.port.eq") { t.Error("port fail") }

	req = &RequestClause{
		IsTLS: &BoolExpr{Value: true, Operator: OpBoolEq},
	}
	if !strings.Contains(req.String(), "req.tls.eq") { t.Error("tls fail") }
	
	// Test other Response fields
	resp := &ResponseClause{
		Body: &StringExpr{Value: "b", Operator: OpEq},
	}
	if !strings.Contains(resp.String(), "resp.body.eq") { t.Error("resp body fail") }
	
	resp = &ResponseClause{
		Length: &IntExpr{Value: 10, Operator: OpIntEq},
	}
	if !strings.Contains(resp.String(), "resp.len.eq") { t.Error("resp len fail") }

    // Test empty
    if (&Query{}).String() != "" { t.Error("empty query") }
    if (&RequestClause{}).String() != "" { t.Error("empty req") }
    if (&ResponseClause{}).String() != "" { t.Error("empty resp") }
}

func TestBoolExpr_String(t *testing.T) {
	be := &BoolExpr{Value: true, Operator: OpBoolEq}
	if s := be.String(); s != "eq:true" {
		t.Errorf("Expected eq:true, got %s", s)
	}
}
