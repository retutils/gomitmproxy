package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/blevesearch/bleve/v2"
	_ "github.com/marcboeker/go-duckdb"
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
			created_at TIMESTAMP
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to init duckdb schema: %w", err)
	}

	// 2. Initialize Bleve Index
	indexPath := filepath.Join(storageDir, "flows.bleve")
	var index bleve.Index
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// Define Mapping
		mapping := bleve.NewIndexMapping()
		
		// Text Field Mapping
		textFieldMapping := bleve.NewTextFieldMapping()
		textFieldMapping.Analyzer = "standard"

		// Document Mapping
		docMapping := bleve.NewDocumentMapping()
		docMapping.AddFieldMappingsAt("Method", textFieldMapping)
		docMapping.AddFieldMappingsAt("URL", textFieldMapping)
		docMapping.AddFieldMappingsAt("ReqBody", textFieldMapping)
		docMapping.AddFieldMappingsAt("ResBody", textFieldMapping)

		// Headers Mapping (Dynamic)
		headerMapping := bleve.NewDocumentMapping()
		headerMapping.Dynamic = true
		headerMapping.DefaultAnalyzer = "standard"
		
		docMapping.AddSubDocumentMapping("ReqHeader", headerMapping)
		docMapping.AddSubDocumentMapping("ResHeader", headerMapping)
		
		// Status Code (Numeric)
		numericFieldMapping := bleve.NewNumericFieldMapping()
		docMapping.AddFieldMappingsAt("Status", numericFieldMapping)

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
		INSERT INTO flows (id, conn_id, method, url, status_code, req_header, req_body, res_header, res_body, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, entry.ID, entry.ConnID, entry.Method, entry.URL, entry.StatusCode, entry.RequestHeader, entry.RequestBody, entry.ResponseHeader, entry.ResponseBody, time.Now())
	
	if err != nil {
		log.Errorf("failed to insert into duckdb: %v", err)
		return err
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

	// 2. Index in Bleve
	// We index relevant fields for search
	doc := struct {
		ID        string
		Method    string
		URL       string
		Status    int
		ReqBody   string
		ResBody   string
		ReqHeader map[string]interface{}
		ResHeader map[string]interface{}
	}{
		ID:        entry.ID,
		Method:    entry.Method,
		URL:       entry.URL,
		Status:    entry.StatusCode,
		ReqBody:   string(entry.RequestBody), // Only suitable for text content really, but bleve handles string
		ResBody:   string(entry.ResponseBody),
		ReqHeader: reqHeaderMap,
		ResHeader: resHeaderMap,
	}

	if err := s.index.Index(entry.ID, doc); err != nil {
		log.Errorf("failed to index in bleve: %v", err)
		return err
	}

	return nil
}

func (s *Service) Search(queryStr string) ([]*FlowEntry, error) {
	// 1. Search in Bleve to get IDs
	query := bleve.NewQueryStringQuery(queryStr)
	searchRequest := bleve.NewSearchRequest(query)
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
