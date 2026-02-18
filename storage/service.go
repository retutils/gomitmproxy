package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/retutils/gomitmproxy/httpql"
	"github.com/retutils/gomitmproxy/proxy"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	db    *sql.DB
	index bleve.Index
}

func NewService(storageDir string) (*Service, error) {
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage dir: %w", err)
	}

	// 1. Initialize DuckDB
	dbPath := filepath.Join(storageDir, "flows.duckdb")
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}

	// Create table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS flows (
			id TEXT PRIMARY KEY,
			conn_id TEXT,
			method TEXT,
			url TEXT,
			status_code INTEGER,
			req_header JSON,
			req_body BLOB,
			res_header JSON,
			res_body BLOB,
			created_at TIMESTAMP,
			has_pii BOOLEAN
		);
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init duckdb schema: %w", err)
	}

	// 2. Initialize Bleve Index
	indexPath := filepath.Join(storageDir, "flows.bleve")
	var index bleve.Index
	if _, statErr := os.Stat(indexPath); os.IsNotExist(statErr) {
		// Define Mapping
		mapping := bleve.NewIndexMapping()

		// Text Field Mapping
		textFieldMapping := bleve.NewTextFieldMapping()
		textFieldMapping.Analyzer = "standard"

		// Lowercase Keyword Mapping for exact matches
		keywordFieldMapping := bleve.NewTextFieldMapping()
		keywordFieldMapping.Analyzer = "keyword"

		// Document Mapping
		docMapping := bleve.NewDocumentMapping()
		docMapping.AddFieldMappingsAt("Method", keywordFieldMapping)
		docMapping.AddFieldMappingsAt("URL", textFieldMapping)
		docMapping.AddFieldMappingsAt("Host", textFieldMapping)
		docMapping.AddFieldMappingsAt("Path", textFieldMapping)
		docMapping.AddFieldMappingsAt("Query", textFieldMapping)
		docMapping.AddFieldMappingsAt("ReqBody", textFieldMapping)
		docMapping.AddFieldMappingsAt("ResBody", textFieldMapping)

		booleanFieldMapping := bleve.NewBooleanFieldMapping()
		docMapping.AddFieldMappingsAt("HasPII", booleanFieldMapping)
		numericFieldMapping := bleve.NewNumericFieldMapping()
		docMapping.AddFieldMappingsAt("Status", numericFieldMapping)
		docMapping.AddFieldMappingsAt("ReqLen", numericFieldMapping)
		docMapping.AddFieldMappingsAt("RespLen", numericFieldMapping)
		docMapping.AddFieldMappingsAt("Port", numericFieldMapping)

		// Headers Mapping (Dynamic)
		headerMapping := bleve.NewDocumentMapping()
		headerMapping.Dynamic = true
		headerMapping.DefaultAnalyzer = "standard"

		docMapping.AddSubDocumentMapping("ReqHeader", headerMapping)
		docMapping.AddSubDocumentMapping("ResHeader", headerMapping)

		mapping.DefaultMapping = docMapping

		index, err = bleve.New(indexPath, mapping)
	} else {
		index, err = bleve.Open(indexPath)
	}
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to open bleve: %w", err)
	}

	return &Service{
		db:    db,
		index: index,
	}, nil
}

func (s *Service) Close() error {
	if s.index != nil {
		s.index.Close()
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Service) Save(f *proxy.Flow) error {
	entry, err := NewFlowEntry(f)
	if err != nil {
		return err
	}

	// 1. Save to DuckDB
	// Note: DuckDB supports standard SQL
	_, err = s.db.Exec(`
		INSERT INTO flows (id, conn_id, method, url, status_code, req_header, req_body, res_header, res_body, created_at, has_pii)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.ConnID, entry.Method, entry.URL, entry.StatusCode, entry.RequestHeader, entry.RequestBody, entry.ResponseHeader, entry.ResponseBody, time.Now(), entry.HasPII)

	if err != nil {
		log.Errorf("failed to insert into duckdb: %v", err)
		return err
	}

	// Save PII detections if present
	if piiData, ok := f.Metadata["pii"]; ok {
		// We expect []addon.PIIFinding, but since Metadata is map[string]interface{}, we might need type assertion or json roundtrip if coming from elsewhere.
		// Since it's in-memory from same process, type assertion should work if we import addon package.
		// But circular dependency proxy <-> addon <-> storage might be issue.
		// Use reflection or mapstructure or just assume it's slice of structs/maps

		// Simple approach: marshal/unmarshal to handle generic interface
		bytes, _ := json.Marshal(piiData)
		var findings []struct {
			Source  string `json:"source"`
			Type    string `json:"type"`
			Snippet string `json:"snippet"`
		}
		json.Unmarshal(bytes, &findings)

		for _, finding := range findings {
			_, err := s.db.Exec(`
				INSERT INTO pii_detections (flow_id, source, type, snippet)
				VALUES (?, ?, ?, ?)
			`, entry.ID, finding.Source, finding.Type, finding.Snippet)
			if err != nil {
				log.Errorf("failed to insert pii detection: %v", err)
			}
		}
	}

	// Unmarshal headers for indexing
	var reqHeaderMap map[string]interface{}
	if err := json.Unmarshal([]byte(entry.RequestHeader), &reqHeaderMap); err != nil {
		reqHeaderMap = make(map[string]interface{})
	}

	var resHeaderMap map[string]interface{}
	if err := json.Unmarshal([]byte(entry.ResponseHeader), &resHeaderMap); err != nil {
		resHeaderMap = make(map[string]interface{})
	}

	// Parse URL for indexing
	parsedURL := f.Request.URL

	// 2. Index in Bleve
	// We index relevant fields for search
	doc := struct {
		ID        string
		Method    string
		URL       string
		Host      string
		Path      string
		Query     string
		Port      int
		Status    int
		ReqLen    int
		RespLen   int
		ReqBody   string
		ResBody   string
		ReqHeader map[string]interface{}
		ResHeader map[string]interface{}
		HasPII    bool
	}{
		ID:     entry.ID,
		Method: entry.Method,
		URL:    entry.URL,
		Host:   parsedURL.Hostname(),
		Path:   parsedURL.Path,
		Query:  parsedURL.RawQuery,
		// Port - extract from Host or URLScheme
		// Status
		Status:    entry.StatusCode,
		ReqLen:    len(entry.RequestBody),
		RespLen:   len(entry.ResponseBody),
		ReqBody:   string(entry.RequestBody),
		ResBody:   string(entry.ResponseBody),
		ReqHeader: reqHeaderMap,
		ResHeader: resHeaderMap,
		HasPII:    entry.HasPII,
	}

	// Try parse port
	if portStr := parsedURL.Port(); portStr != "" {
		fmt.Sscanf(portStr, "%d", &doc.Port)
	}

	if err := s.index.Index(entry.ID, doc); err != nil {
		log.Errorf("failed to index in bleve: %v", err)
		return err
	}

	return nil
}

func (s *Service) Search(queryStr string) ([]*FlowEntry, error) {
	// 1. Search in Bleve to get IDs
	var indexQuery query.Query

	// Try parsing as HTTPQL
	l := httpql.NewLexer(queryStr)
	p := httpql.NewParser(l)
	qlQuery, err := p.ParseQuery()
	if err == nil {
		indexQuery = BuildBleveQuery(qlQuery)
	} else {
		// Fallback to standard Bleve Query String
		indexQuery = bleve.NewQueryStringQuery(queryStr)
	}

	searchRequest := bleve.NewSearchRequest(indexQuery)
	searchResult, err := s.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	if searchResult.Total == 0 {
		return []*FlowEntry{}, nil
	}

	ids := make([]string, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		ids = append(ids, hit.ID)
	}

	// 2. Retrieve from DuckDB
	// DuckDB doesn't support array parameter easily in standard sql driver, so we loop or build query
	// Using loop for simplicity for now, optimal way is likely WHERE id IN (...)

	// Construct WHERE IN clause safely?
	// For simplicity in this iteration, let's just loop. It's not efficient for large result sets but works.

	results := make([]*FlowEntry, 0, len(ids))
	for _, id := range ids {
		row := s.db.QueryRow(`
			SELECT id, conn_id, method, url, status_code, req_header, req_body, res_header, res_body
			FROM flows WHERE id = ?
		`, id)

		var e FlowEntry
		var reqBody, resBody []byte
		var reqHeader, resHeader interface{}

		err := row.Scan(&e.ID, &e.ConnID, &e.Method, &e.URL, &e.StatusCode, &reqHeader, &reqBody, &resHeader, &resBody)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, err
		}
		e.RequestBody = reqBody
		e.ResponseBody = resBody

		// Convert headers back to string
		if reqHeader != nil {
			bytes, _ := json.Marshal(reqHeader)
			e.RequestHeader = string(bytes)
		}
		if resHeader != nil {
			bytes, _ := json.Marshal(resHeader)
			e.ResponseHeader = string(bytes)
		}

		results = append(results, &e)
	}

	return results, nil
}
