package proxy

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"

	uuid "github.com/satori/go.uuid"
)

// flow http request
type Request struct {
	Method string
	URL    *url.URL
	Proto  string
	Header http.Header
	Body   []byte

	raw *http.Request
}

func NewRequest(req *http.Request) *Request {
	return &Request{
		Method: req.Method,
		URL:    req.URL,
		Proto:  req.Proto,
		Header: req.Header,
		raw:    req,
	}
}

func (r *Request) SetRaw(req *http.Request) {
	r.raw = req
}

func (r *Request) Raw() *http.Request {
	return r.raw
}

func (req *Request) MarshalJSON() ([]byte, error) {
	type requestJSON struct {
		Method string      `json:"method"`
		URL    string      `json:"url"`
		Proto  string      `json:"proto"`
		Header http.Header `json:"header"`
	}
	urlStr := ""
	if req.URL != nil {
		urlStr = req.URL.String()
	}
	return json.Marshal(&requestJSON{
		Method: req.Method,
		URL:    urlStr,
		Proto:  req.Proto,
		Header: req.Header,
	})
}

func (req *Request) UnmarshalJSON(data []byte) error {
	r := make(map[string]interface{})
	err := json.Unmarshal(data, &r)
	if err != nil {
		return err
	}

	rawurl, ok := r["url"].(string)
	if !ok {
		return errors.New("url parse error")
	}
	u, err := url.Parse(rawurl)
	if err != nil {
		return err
	}

	rawheader, ok := r["header"].(map[string]interface{})
	if !ok {
		return errors.New("rawheader parse error")
	}

	header := make(map[string][]string)
	for k, v := range rawheader {
		vals, ok := v.([]interface{})
		if !ok {
			return errors.New("header parse error")
		}

		svals := make([]string, 0)
		for _, val := range vals {
			sval, ok := val.(string)
			if !ok {
				return errors.New("header parse error")
			}
			svals = append(svals, sval)
		}
		header[k] = svals
	}

	getString := func(key string) string {
		if v, ok := r[key].(string); ok {
			return v
		}
		return ""
	}

	*req = Request{
		Method: getString("method"),
		URL:    u,
		Proto:  getString("proto"),
		Header: header,
	}
	return nil
}

// flow http response
type Response struct {
	StatusCode int         `json:"statusCode"`
	Header     http.Header `json:"header"`
	Body       []byte      `json:"-"`
	BodyReader io.Reader

	close bool // connection close
}

// flow
type Flow struct {
	Id          uuid.UUID    `json:"id"`
	ConnContext *ConnContext `json:"-"`
	Request     *Request     `json:"request"`
	Response    *Response    `json:"response"`

	// https://docs.mitmproxy.org/stable/overview-features/#streaming
	// 如果为 true，则不缓冲 Request.Body 和 Response.Body，且不进入之后的 Addon.Request 和 Addon.Response
	Stream            bool                   `json:"-"`
	UseSeparateClient bool                   `json:"-"` // use separate http client to send http request
	done              chan struct{}          `json:"-"`

	// Metadata to pass data between addons. Not persisted by default unless handled by storage addon.
	Metadata map[string]interface{} `json:"-"`
}

func NewFlow() *Flow {
	return &Flow{
		Id:       uuid.NewV4(),
		done:     make(chan struct{}),
		Metadata: make(map[string]interface{}),
	}
}

func (f *Flow) Done() <-chan struct{} {
	return f.done
}

func (f *Flow) Finish() {
	close(f.done)
}

func (f *Flow) MarshalJSON() ([]byte, error) {
	type Alias Flow
	return json.Marshal((*Alias)(f))
}

