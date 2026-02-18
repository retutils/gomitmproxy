package httpql

import (
	"fmt"
	"strconv"
)

type Parser struct {
	l       *Lexer
	curTok  Token
	peekTok Token
}

func NewParser(l *Lexer) *Parser {
	p := &Parser{l: l}
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curTok = p.peekTok
	p.peekTok = p.l.NextToken()
}

func (p *Parser) ParseQuery() (*Query, error) {
	// Root query: Clause (OR Clause)* -> handled by expression parsing
	// Since HTTPQL is expression based:
	// Query = Expr
	// Expr = Term { OR Term }
	// Term = Factor { AND Factor }
	// Factor = ( Expr ) | Clause

	return p.parseOr()
}

func (p *Parser) parseOr() (*Query, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.curTok.Type == TOKEN_OR {
		p.nextToken() // consume OR
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		// Create OR node
		left = &Query{
			Or: []*Query{left, right},
		}
	}
	return left, nil
}

func (p *Parser) parseAnd() (*Query, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}

	for p.curTok.Type == TOKEN_AND {
		p.nextToken() // consume AND
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		left = &Query{
			And: []*Query{left, right},
		}
	}
	return left, nil
}

func (p *Parser) parseFactor() (*Query, error) {
	if p.curTok.Type == TOKEN_LPAREN {
		p.nextToken() // (
		q, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.curTok.Type != TOKEN_RPAREN {
			return nil, fmt.Errorf("expected ), got %v", p.curTok.Literal)
		}
		p.nextToken() // )
		return q, nil
	}

	// It must be a clause: req.field.op:val
	return p.parseClause()
}

func (p *Parser) parseClause() (*Query, error) {
	// Expect: req/resp . field . op : val

	namespace := p.curTok.Literal
	if namespace != "req" && namespace != "resp" {
		return nil, fmt.Errorf("expected req/resp, got %s", namespace)
	}
	p.nextToken()

	if p.curTok.Type != TOKEN_DOT {
		return nil, fmt.Errorf("expected ., got %s", p.curTok.Literal)
	}
	p.nextToken()

	field := p.curTok.Literal
	p.nextToken()

	if p.curTok.Type != TOKEN_DOT {
		return nil, fmt.Errorf("expected . after field, got %s", p.curTok.Literal)
	}
	p.nextToken()

	op := p.curTok.Literal
	p.nextToken()

	if p.curTok.Type != TOKEN_COLON {
		return nil, fmt.Errorf("expected :, got %s", p.curTok.Literal)
	}
	p.nextToken()

	val := p.curTok.Literal
	p.nextToken()

	if namespace == "req" {
		return p.buildReqClause(field, op, val)
	} else {
		return p.buildRespClause(field, op, val)
	}
}

func (p *Parser) buildReqClause(field, op, val string) (*Query, error) {
	clause := &RequestClause{}

	switch field {
	case "method":
		clause.Method = &StringExpr{Value: val, Operator: StringOp(op)}
	case "host":
		clause.Host = &StringExpr{Value: val, Operator: StringOp(op)}
	case "path":
		clause.Path = &StringExpr{Value: val, Operator: StringOp(op)}
	case "query":
		clause.Query = &StringExpr{Value: val, Operator: StringOp(op)}
	case "port":
		expr, err := parseIntExpr(val, op)
		if err != nil {
			return nil, err
		}
		clause.Port = expr
	case "body", "raw", "ext":
		clause.Body = &StringExpr{Value: val, Operator: StringOp(op)}
	default:
		return nil, fmt.Errorf("unknown req field: %s", field)
	}

	return &Query{Req: clause}, nil
}

func (p *Parser) buildRespClause(field, op, val string) (*Query, error) {
	clause := &ResponseClause{}

	switch field {
	case "code":
		expr, err := parseIntExpr(val, op)
		if err != nil {
			return nil, err
		}
		clause.StatusCode = expr
	case "len":
		expr, err := parseIntExpr(val, op)
		if err != nil {
			return nil, err
		}
		clause.Length = expr
	case "body", "raw":
		clause.Body = &StringExpr{Value: val, Operator: StringOp(op)}
	default:
		return nil, fmt.Errorf("unknown resp field: %s", field)
	}

	return &Query{Resp: clause}, nil
}

func parseIntExpr(val, op string) (*IntExpr, error) {
	v, err := strconv.Atoi(val)
	if err != nil {
		return nil, err
	}
	return &IntExpr{Value: v, Operator: IntOp(op)}, nil
}
