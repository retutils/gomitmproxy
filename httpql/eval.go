package httpql

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/retutils/gomitmproxy/proxy"
)

func (q *Query) Eval(f *proxy.Flow) bool {
	if q == nil {
		return true
	}
	if q.And != nil {
		for _, sub := range q.And {
			if !sub.Eval(f) {
				return false
			}
		}
		return true
	}
	if q.Or != nil {
		for _, sub := range q.Or {
			if sub.Eval(f) {
				return true
			}
		}
		return false
	}
	if q.Req != nil {
		return q.Req.Eval(f)
	}
	if q.Resp != nil {
		return q.Resp.Eval(f)
	}
	return true
}

func (r *RequestClause) Eval(f *proxy.Flow) bool {
	if f.Request == nil {
		return false
	}
	if r.Method != nil && !r.Method.Eval(f.Request.Method) {
		return false
	}
	if r.Host != nil && !r.Host.Eval(f.Request.URL.Hostname()) {
		return false
	}
	if r.Path != nil && !r.Path.Eval(f.Request.URL.Path) {
		return false
	}
	if r.Query != nil && !r.Query.Eval(f.Request.URL.RawQuery) {
		return false
	}
	if r.Body != nil {
		body, err := f.Request.DecodedBody()
		if err == nil && !r.Body.Eval(string(body)) {
			return false
		}
	}
	if r.Port != nil {
		port, _ := strconv.Atoi(f.Request.URL.Port())
		if port == 0 {
			if f.Request.URL.Scheme == "https" {
				port = 443
			} else {
				port = 80
			}
		}
		if !r.Port.Eval(port) {
			return false
		}
	}
	if r.IsTLS != nil {
		isTls := f.Request.URL.Scheme == "https"
		if !r.IsTLS.Eval(isTls) {
			return false
		}
	}
	return true
}

func (r *ResponseClause) Eval(f *proxy.Flow) bool {
	if f.Response == nil {
		return false
	}
	if r.StatusCode != nil && !r.StatusCode.Eval(f.Response.StatusCode) {
		return false
	}
	if r.Body != nil {
		body, err := f.Response.DecodedBody()
		if err == nil && !r.Body.Eval(string(body)) {
			return false
		}
	}
	if r.Length != nil {
		body, err := f.Response.DecodedBody()
		if err == nil && !r.Length.Eval(len(body)) {
			return false
		}
	}
	return true
}

func (s *StringExpr) Eval(val string) bool {
	switch s.Operator {
	case OpEq:
		return val == s.Value
	case OpNe:
		return val != s.Value
	case OpCont:
		return strings.Contains(val, s.Value)
	case OpNCont:
		return !strings.Contains(val, s.Value)
	case OpLike:
		// simple wildcard
		matched, _ := regexp.MatchString(ConvertLikeToRegex(s.Value), val)
		return matched
	case OpNLike:
		matched, _ := regexp.MatchString(ConvertLikeToRegex(s.Value), val)
		return !matched
	case OpRegex:
		matched, _ := regexp.MatchString(s.Value, val)
		return matched
	case OpNRegex:
		matched, _ := regexp.MatchString(s.Value, val)
		return !matched
	}
	return false
}

func ConvertLikeToRegex(like string) string {
	// Support both SQL-style (%) and Glob-style (*) wildcards
	regex := regexp.QuoteMeta(like)
	// Glob style
	regex = strings.ReplaceAll(regex, "\\*", ".*")
	regex = strings.ReplaceAll(regex, "\\?", ".")
	// SQL style (optional, but good for compatibility if expected)
	regex = strings.ReplaceAll(regex, "%", ".*")
	regex = strings.ReplaceAll(regex, "_", ".")
	return "^" + regex + "$"
}

func (i *IntExpr) Eval(val int) bool {
	switch i.Operator {
	case OpIntEq:
		return val == i.Value
	case OpIntNe:
		return val != i.Value
	case OpIntGt:
		return val > i.Value
	case OpIntGte:
		return val >= i.Value
	case OpIntLt:
		return val < i.Value
	case OpIntLte:
		return val <= i.Value
	}
	return false
}

func (b *BoolExpr) Eval(val bool) bool {
	switch b.Operator {
	case OpBoolEq:
		return val == b.Value
	case OpBoolNe:
		return val != b.Value
	}
	return false
}
