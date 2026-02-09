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
		// Contains -> Wildcard *val*
		wq := query.NewWildcardQuery("*" + s.Value + "*")
		wq.SetField(field)
		return wq

	case httpql.OpNCont:
		q := buildStringQuery(field, &httpql.StringExpr{Value: s.Value, Operator: httpql.OpCont})
		bq := query.NewBooleanQuery(nil, nil, nil)
		bq.AddMustNot(q)
		return bq

	case httpql.OpLike:
		// Convert SQL Like to wildcard if simple, or Regex
		// Simple conversion: % -> *
		val := s.Value
		// val = strings.ReplaceAll(val, "%", "*") // Bleve wildcard uses * 
		// Actually best to use Regexp query for full LIKE support
		regex := convertLikeToRegex(val)
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
	inclusive := true

	switch i.Operator {
	case httpql.OpIntEq:
		nq := query.NewNumericRangeQuery(&val, &val)
		nq.InclusiveMin = &inclusive
		nq.InclusiveMax = &inclusive
		nq.SetField(field)
		return nq
	case httpql.OpIntNe:
		q := buildIntQuery(field, &httpql.IntExpr{Value: i.Value, Operator: httpql.OpIntEq})
		bq := query.NewBooleanQuery(nil, nil, nil)
		bq.AddMustNot(q)
		return bq
	case httpql.OpIntGt:
		nq := query.NewNumericRangeQuery(&val, nil)
		nq.InclusiveMin = &msgFalse
		nq.SetField(field)
		return nq
	case httpql.OpIntGte:
		nq := query.NewNumericRangeQuery(&val, nil)
		nq.InclusiveMin = &inclusive
		nq.SetField(field)
		return nq
	case httpql.OpIntLt:
		nq := query.NewNumericRangeQuery(nil, &val)
		nq.InclusiveMax = &msgFalse
		nq.SetField(field)
		return nq
	case httpql.OpIntLte:
		nq := query.NewNumericRangeQuery(nil, &val)
		nq.InclusiveMax = &inclusive
		nq.SetField(field)
		return nq
	}
	return query.NewMatchAllQuery()
}

var msgFalse = false
var msgTrue = true

func convertLikeToRegex(like string) string {
	return httpql.ConvertLikeToRegex(like)
}
