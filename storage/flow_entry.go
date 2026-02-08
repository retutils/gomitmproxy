package storage

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/retutils/gomitmproxy/proxy"
	uuid "github.com/satori/go.uuid"
)

// FlowEntry represents a stored HTTP flow optimized for database storage
type FlowEntry struct {
	ID             string    `json:"id"`
	ConnID         string    `json:"conn_id"`
	Method         string    `json:"method"`
	URL            string    `json:"url"`
	Proto          string    `json:"proto"`
	StatusCode     int       `json:"status_code"`
	RequestHeader  string    `json:"request_header"`  // JSON string
	RequestBody    []byte    `json:"request_body"`
	ResponseHeader string    `json:"response_header"` // JSON string
	ResponseBody   []byte    `json:"response_body"`
	BodyIsText     bool      `json:"body_is_text"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	DurationMs     int64     `json:"duration_ms"`
}

// NewFlowEntry converts a proxy.Flow to a storage-ready FlowEntry
func NewFlowEntry(f *proxy.Flow) (*FlowEntry, error) {
	reqHeaderJSON, err := json.Marshal(f.Request.Header)
	if err != nil {
		reqHeaderJSON = []byte("{}")
	}

	resHeaderJSON := []byte("{}")
	statusCode := 0
	var resBody []byte
	isText := false

	if f.Response != nil {
		resHeaderJSON, err = json.Marshal(f.Response.Header)
		if err != nil {
			resHeaderJSON = []byte("{}")
		}
		statusCode = f.Response.StatusCode
		resBody, _ = f.Response.DecodedBody()
		isText = f.Response.IsTextContentType()
	}

	reqBody, _ := f.Request.DecodedBody()

	// Approximate duration if start/end times aren't explicitly tracked in Flow
	// For now we use current time as end time
	endTime := time.Now()
	startTime := endTime // TODO: Flow doesn't expose timing yet, using current time placeholder

	return &FlowEntry{
		ID:             f.Id.String(),
		ConnID:         f.ConnContext.Id().String(),
		Method:         f.Request.Method,
		URL:            f.Request.URL.String(),
		Proto:          f.Request.Proto,
		StatusCode:     statusCode,
		RequestHeader:  string(reqHeaderJSON),
		RequestBody:    reqBody,
		ResponseHeader: string(resHeaderJSON),
		ResponseBody:   resBody,
		BodyIsText:     isText,
		StartTime:      startTime,
		EndTime:        endTime,
		DurationMs:     0,
	}, nil
}

// ToProxyFlow converts a FlowEntry back to a proxy.Flow (partial implementation for display)
func (e *FlowEntry) ToProxyFlow() (*proxy.Flow, error) {
	// Reconstruct Request Headers
	var reqHeader http.Header
	if err := json.Unmarshal([]byte(e.RequestHeader), &reqHeader); err != nil {
		return nil, err
	}

	// Reconstruct Response Headers
	var resHeader http.Header
	if err := json.Unmarshal([]byte(e.ResponseHeader), &resHeader); err != nil {
		return nil, err
	}

	id, err := uuid.FromString(e.ID)
	if err != nil {
		return nil, err
	}

	// NOTE: This reconstructs a display-only Flow, not a functional one for replay
	// URL parsing omitted for brevity, assuming simple string reconstruction
	
	return &proxy.Flow{
		Id: id,
		// Request and Response would need full reconstruction if needed
	}, nil
}
