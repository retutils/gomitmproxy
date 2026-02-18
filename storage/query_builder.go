package storage

import (
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/retutils/gomitmproxy/httpql"
)

func BuildBleveQuery(q *httpql.Query) query.Query {
	if q == nil {
		return query.NewMatchAllQuery()
	}

	if len(q.And) > 0 || len(q.Or) > 0 {
		bq := query.NewBooleanQuery(nil, nil, nil)

		if len(q.And) == 2 {
			left := BuildBleveQuery(q.And[0])
			right := BuildBleveQuery(q.And[1])
			bq.AddMust(left, right)
		}

		if len(q.Or) == 2 {
			left := BuildBleveQuery(q.Or[0])
			right := BuildBleveQuery(q.Or[1])
			// Bleve BooleanQuery with Should mimics OR
			bq.AddShould(left, right)
			bq.SetMinShould(1)
		}
		return bq
	}

	if q.Req != nil {
		return buildReqQuery(q.Req)
	}

	if q.Resp != nil {
		return buildRespQuery(q.Resp)
	}

	return query.NewMatchAllQuery()
}

func buildReqQuery(r *httpql.RequestClause) query.Query {
	bq := query.NewBooleanQuery(nil, nil, nil)

	if r.Method != nil {
		bq.AddMust(buildStringQuery("Method", r.Method))
	}
	if r.Host != nil {
		bq.AddMust(buildStringQuery("Host", r.Host))
	}
	if r.Path != nil {
		bq.AddMust(buildStringQuery("Path", r.Path))
	}
	if r.Query != nil {
		bq.AddMust(buildStringQuery("Query", r.Query))
	}
	if r.Body != nil {
		bq.AddMust(buildStringQuery("ReqBody", r.Body))
	}
	if r.Port != nil {
		bq.AddMust(buildIntQuery("Port", r.Port))
	}
	// TODO: TLS?

	return bq
}

func buildRespQuery(r *httpql.ResponseClause) query.Query {
	bq := query.NewBooleanQuery(nil, nil, nil)

	if r.StatusCode != nil {
		bq.AddMust(buildIntQuery("Status", r.StatusCode))
	}
	if r.Body != nil {
		bq.AddMust(buildStringQuery("ResBody", r.Body))
	}
	if r.Length != nil {
		bq.AddMust(buildIntQuery("RespLen", r.Length))
	}

	return bq
}

func buildStringQuery(field string, s *httpql.StringExpr) query.Query {
	switch s.Operator {
	case httpql.OpEq:
		// Exact match (Keyword analyzer) or Term query
		// For fields analyzed with "standard", TermQuery matches tokens.
		// For "keyword" analyzer (Method), TermQuery matches exact string.
		// For others, MatchQuery is safer for text.
		if field == "Method" {
			tq := query.NewTermQuery(s.Value)
			tq.SetField(field)
			return tq
		}
		mq := query.NewMatchQuery(s.Value)
		mq.SetField(field)
		return mq

	case httpql.OpNe:
		// Negation
		q := buildStringQuery(field, &httpql.StringExpr{Value: s.Value, Operator: httpql.OpEq})
		bq := query.NewBooleanQuery(nil, nil, nil)
		bq.AddMustNot(q)
		return bq

	case httpql.OpCont:
		// Contains -> Wildcard is bad for analyzed text.
		// If field is "Method" (keyword), Wildcard is fine.
		// If field is "ReqBody" or "ResBody" (standard), Wildcard *foo* only matches if 'foo' is a token.
		// Better approach for standard text: MatchQuery (matches tokens) or specialized logic.
		// For consistency with "Contains", if it's a phrase, we probably want MatchPhrase.

		if field == "Method" {
			wq := query.NewWildcardQuery("*" + s.Value + "*")
			wq.SetField(field)
			return wq
		}

		// For text fields, "contains" usually means the terms are present.
		// We use MatchPhrase to ensure order if multiple words.
		mpq := query.NewMatchPhraseQuery(s.Value)
		mpq.SetField(field)
		return mpq

	case httpql.OpNCont:
		q := buildStringQuery(field, &httpql.StringExpr{Value: s.Value, Operator: httpql.OpCont})
		bq := query.NewBooleanQuery(nil, nil, nil)
		bq.AddMustNot(q)
		return bq

	case httpql.OpLike:
		// Convert SQL Like to wildcard if simple, or Regex
		regex := httpql.ConvertLikeToRegex(s.Value)
		rq := query.NewRegexpQuery(regex)
		rq.SetField(field)
		return rq

	case httpql.OpRegex:
		rq := query.NewRegexpQuery(s.Value)
		rq.SetField(field)
		return rq
	}
	return query.NewMatchAllQuery()
}

func buildIntQuery(field string, i *httpql.IntExpr) query.Query {
	val := float64(i.Value)

	// Handle NE (Not Equals) separately as it requires boolean logic
	if i.Operator == httpql.OpIntNe {
		q := buildIntQuery(field, &httpql.IntExpr{Value: i.Value, Operator: httpql.OpIntEq})
		bq := query.NewBooleanQuery(nil, nil, nil)
		bq.AddMustNot(q)
		return bq
	}

	var min, max *float64
	var minIncl, maxIncl *bool

	switch i.Operator {
	case httpql.OpIntEq:
		min, max = &val, &val
		minIncl, maxIncl = &msgTrue, &msgTrue
	case httpql.OpIntGt:
		min = &val
		minIncl = &msgFalse
	case httpql.OpIntGte:
		min = &val
		minIncl = &msgTrue
	case httpql.OpIntLt:
		max = &val
		maxIncl = &msgFalse
	case httpql.OpIntLte:
		max = &val
		maxIncl = &msgTrue
	default:
		return query.NewMatchAllQuery()
	}

	nq := query.NewNumericRangeQuery(min, max)
	if minIncl != nil {
		nq.InclusiveMin = minIncl
	}
	if maxIncl != nil {
		nq.InclusiveMax = maxIncl
	}
	nq.SetField(field)
	return nq
}

var msgFalse = false
var msgTrue = true
