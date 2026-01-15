package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/celerix-dev/celerix-store/pkg/sdk"
	"github.com/celerix/depot/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func setupTestHandler(t *testing.T) (*Handler, string, func()) {
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "depot-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dataDir := filepath.Join(tempDir, "data")
	storageDir := filepath.Join(tempDir, "uploads")
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(storageDir, 0755)

	store, err := sdk.New(dataDir)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	h := &Handler{
		Store:            store,
		StorageDir:       storageDir,
		AdminSecret:      "test-secret",
		VersionConfig:    []byte(`{"version": "1.0.0-test"}`),
		CelerixNamespace: uuid.New(),
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return h, storageDir, cleanup
}

func TestGetVersion(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.GetVersion(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["version"] != "1.0.0-test" {
		t.Errorf("expected version 1.0.0-test, got %s", resp["version"])
	}
}

func TestPersonaFlow(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.Default()
	router.POST("/persona/name", h.UpdateClientName)
	router.GET("/persona", h.GetPersona)
	router.POST("/persona/admin", h.ActivateAdmin)

	// 1. Update Client Name (Initial)
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"name": "Test User"}`)
	req, _ := http.NewRequest("POST", "/persona/name", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", "initial-id")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UpdateClientName failed: %v", w.Body.String())
	}

	var nameResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &nameResp)
	clientID := nameResp["id"]
	recoveryCode := nameResp["recovery_code"]

	if clientID == "" || recoveryCode == "" {
		t.Errorf("Missing ID or recovery code in response")
	}

	// 2. Get Persona
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/persona", nil)
	req.Header.Set("X-Client-ID", clientID)
	router.ServeHTTP(w, req)

	var personaResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &personaResp)
	if personaResp["name"] != "Test User" {
		t.Errorf("expected name Test User, got %v", personaResp["name"])
	}
	if personaResp["persona"] != "client" {
		t.Errorf("expected persona client, got %v", personaResp["persona"])
	}

	// 3. Activate Admin
	w = httptest.NewRecorder()
	adminBody := bytes.NewBufferString(`{"secret": "test-secret"}`)
	req, _ = http.NewRequest("POST", "/persona/admin", adminBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", clientID)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ActivateAdmin failed: %v", w.Body.String())
	}

	// 4. Verify Admin Persona
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/persona", nil)
	req.Header.Set("X-Client-ID", clientID)
	router.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &personaResp)
	if personaResp["persona"] != "admin" {
		t.Errorf("expected persona admin, got %v", personaResp["persona"])
	}

	// 5. Check if data is stored in the correct persona
	// In the new implementation, client data is still in SystemPersona
	// to allow global listing and searching across personas.
	val, err := h.Store.Get(db.SystemPersona, "depot", "client:"+clientID)
	if err != nil || val == nil {
		t.Errorf("expected client data in system persona, but not found")
	}
}

func TestClientManagement(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.Default()
	router.POST("/persona/name", h.UpdateClientName)
	router.GET("/persona", h.GetPersona)
	router.POST("/persona/admin", h.ActivateAdmin)
	router.GET("/clients", h.ListClients)
	router.PUT("/clients/:id", h.UpdateClient)
	router.DELETE("/clients/:id", h.DeleteClient)

	// 1. Setup Admin
	w := httptest.NewRecorder()
	body := bytes.NewBufferString(`{"name": "Admin User"}`)
	req, _ := http.NewRequest("POST", "/persona/name", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", "admin-id")
	router.ServeHTTP(w, req)

	var adminResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &adminResp)
	adminID := adminResp["id"]

	w = httptest.NewRecorder()
	adminBody := bytes.NewBufferString(`{"secret": "test-secret"}`)
	req, _ = http.NewRequest("POST", "/persona/admin", adminBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", adminID)
	router.ServeHTTP(w, req)

	// 2. Setup another client
	w = httptest.NewRecorder()
	body = bytes.NewBufferString(`{"name": "Other User"}`)
	req, _ = http.NewRequest("POST", "/persona/name", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", "other-id")
	router.ServeHTTP(w, req)

	var otherResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &otherResp)
	otherID := otherResp["id"]

	// 3. List Clients (as Admin)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/clients", nil)
	req.Header.Set("X-Client-ID", adminID)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListClients failed: %v", w.Body.String())
	}

	var clients []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &clients)
	if len(clients) < 2 {
		t.Errorf("expected at least 2 clients, got %d", len(clients))
	}

	// 4. Update Client (as Admin)
	w = httptest.NewRecorder()
	updateBody := bytes.NewBufferString(`{"name": "Updated Name", "recovery_code": "NEWCODE1", "is_admin": false}`)
	req, _ = http.NewRequest("PUT", "/clients/"+otherID, updateBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", adminID)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UpdateClient failed: %v", w.Body.String())
	}

	// 5. Delete Client (as Admin)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/clients/"+otherID, nil)
	req.Header.Set("X-Client-ID", adminID)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("DeleteClient failed: %v", w.Body.String())
	}
}

func TestFileUploadAndList(t *testing.T) {
	h, _, cleanup := setupTestHandler(t)
	defer cleanup()

	router := gin.Default()
	router.POST("/upload", h.UploadFile)
	router.GET("/files", h.ListFiles)

	clientID := "test-client-id"

	// 1. Upload File
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("hello world"))
	writer.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Client-ID", clientID)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Upload failed: %v", w.Body.String())
	}

	var uploadResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &uploadResp)
	fileID := uploadResp["id"].(string)

	// 2. List Files
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/files", nil)
	req.Header.Set("X-Client-ID", clientID)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("List files failed: %v", w.Body.String())
	}

	var listResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &listResp)
	files := listResp["files"].([]interface{})
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}

	firstFile := files[0].(map[string]interface{})
	if firstFile["id"] != fileID {
		t.Errorf("expected file ID %s, got %v", fileID, firstFile["id"])
	}

	// 3. Verify file is in the persona's store
	val, err := h.Store.Get(clientID, "depot", "file:"+fileID)
	if err != nil || val == nil {
		t.Errorf("expected file record in persona %s, but not found", clientID)
	}

	// 4. Verify file is NOT in system store
	val, err = h.Store.Get(db.SystemPersona, "depot", "file:"+fileID)
	if err == nil && val != nil {
		t.Errorf("expected file record NOT to be in system persona")
	}

	// 5. Verify UpdateFileRecord moves record if owner changes
	// We need an admin request context or just call DB directly
	newOwnerID := "new-owner-id"
	err = db.UpdateFileRecord(h.Store, fileID, "updated.txt", newOwnerID)
	if err != nil {
		t.Errorf("UpdateFileRecord failed: %v", err)
	}

	// Should be in new persona
	val, err = h.Store.Get(newOwnerID, "depot", "file:"+fileID)
	if err != nil || val == nil {
		t.Errorf("expected file record in NEW persona %s, but not found", newOwnerID)
	}

	// Should NOT be in old persona
	val, err = h.Store.Get(clientID, "depot", "file:"+fileID)
	if err == nil && val != nil {
		t.Errorf("expected file record NOT to be in OLD persona anymore")
	}
}
