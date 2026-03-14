package toolsdata

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andresbott/etna/internal/toolsdata"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testHandler(t *testing.T) *Handler {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("unable to open sqlite: %v", err)
	}
	store, err := toolsdata.NewStore(db)
	if err != nil {
		t.Fatalf("unable to create store: %v", err)
	}
	return &Handler{Store: store}
}

func TestListCases_Empty(t *testing.T) {
	h := testHandler(t)
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v0/tools/portfolio-simulator/cases", nil)
	h.ListCases("portfolio-simulator").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var items []casePayload
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatalf("unable to decode response: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestCreateCase_Success(t *testing.T) {
	h := testHandler(t)
	body := `{"name":"Test Case","description":"desc","expectedAnnualReturn":5.0,"params":{"x":1}}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v0/tools/portfolio-simulator/cases", bytes.NewBufferString(body))
	h.CreateCase("portfolio-simulator").ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var p casePayload
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("unable to decode response: %v", err)
	}
	if p.ID == 0 {
		t.Error("expected non-zero id")
	}
	if p.Name != "Test Case" {
		t.Errorf("expected name %q, got %q", "Test Case", p.Name)
	}
	if p.ToolType != "portfolio-simulator" {
		t.Errorf("expected toolType %q, got %q", "portfolio-simulator", p.ToolType)
	}
}

func TestCreateCase_ValidationError(t *testing.T) {
	h := testHandler(t)
	body := `{"name":"","description":"desc"}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v0/tools/portfolio-simulator/cases", bytes.NewBufferString(body))
	h.CreateCase("portfolio-simulator").ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestGetCase_NotFound(t *testing.T) {
	h := testHandler(t)
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v0/tools/portfolio-simulator/cases/99999", nil)
	h.GetCase("portfolio-simulator", 99999).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestUpdateCase_Success(t *testing.T) {
	h := testHandler(t)

	// Create first
	body := `{"name":"Original","description":"desc","expectedAnnualReturn":5.0,"params":{"x":1}}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(body))
	h.CreateCase("portfolio-simulator").ServeHTTP(rec, req)

	var created casePayload
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// Update
	updateBody := `{"name":"Updated","description":"new desc","expectedAnnualReturn":7.0,"params":{"x":2}}`
	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("PUT", "/", bytes.NewBufferString(updateBody))
	h.UpdateCase("portfolio-simulator", created.ID).ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var updated casePayload
	_ = json.Unmarshal(rec2.Body.Bytes(), &updated)
	if updated.Name != "Updated" {
		t.Errorf("expected name %q, got %q", "Updated", updated.Name)
	}
}

func TestUpdateCase_NotFound(t *testing.T) {
	h := testHandler(t)
	body := `{"name":"X","description":""}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/", bytes.NewBufferString(body))
	h.UpdateCase("portfolio-simulator", 99999).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestListCases_AfterCreate(t *testing.T) {
	h := testHandler(t)

	// Create a case
	body := `{"name":"Listed Case","description":"desc","expectedAnnualReturn":4.0,"params":{}}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(body))
	h.CreateCase("portfolio-simulator").ServeHTTP(rec, req)

	// List
	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/", nil)
	h.ListCases("portfolio-simulator").ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec2.Code)
	}
	var items []casePayload
	_ = json.Unmarshal(rec2.Body.Bytes(), &items)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "Listed Case" {
		t.Errorf("expected name %q, got %q", "Listed Case", items[0].Name)
	}
}

func TestDeleteCase_Success(t *testing.T) {
	h := testHandler(t)

	// Create first
	body := `{"name":"To Delete","description":"","expectedAnnualReturn":3.0,"params":{}}`
	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/", bytes.NewBufferString(body))
	h.CreateCase("portfolio-simulator").ServeHTTP(rec, req)

	var created casePayload
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// Delete
	rec2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("DELETE", "/", nil)
	h.DeleteCase("portfolio-simulator", created.ID).ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec2.Code)
	}
}
