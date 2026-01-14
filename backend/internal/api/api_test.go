package api

import (
	"bytes"
	"encoding/json"
	"github.com/celerix/depot/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPersonaRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dbPath := "./test_recover.db"
	storageDir := "./test_recover_uploads"
	defer os.Remove(dbPath)
	defer os.RemoveAll(storageDir)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer database.Close()

	h := &Handler{
		DB:          database,
		StorageDir:  storageDir,
		AdminSecret: "supersecret",
	}

	r := gin.New()
	r.GET("/persona", h.GetPersona)
	r.POST("/persona/name", h.UpdateClientName)
	r.POST("/persona/recover", h.RecoverPersona)

	// 1. Create a client
	clientID := uuid.New().String()
	name := "Recovery Test User"
	body, _ := json.Marshal(gin.H{"name": name})
	req, _ := http.NewRequest("POST", "/persona/name", strings.NewReader(string(body)))
	req.Header.Set("X-Client-ID", clientID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Update client name failed: %s", w.Body.String())
	}

	var updateResp struct {
		RecoveryCode string `json:"recovery_code"`
	}
	json.Unmarshal(w.Body.Bytes(), &updateResp)
	recoveryCode := updateResp.RecoveryCode
	if recoveryCode == "" {
		t.Fatal("Recovery code not returned in UpdateClientName")
	}

	// 2. Recover using the code
	recoverBody, _ := json.Marshal(gin.H{"code": recoveryCode})
	reqRec, _ := http.NewRequest("POST", "/persona/recover", strings.NewReader(string(recoverBody)))
	wRec := httptest.NewRecorder()
	r.ServeHTTP(wRec, reqRec)

	if wRec.Code != http.StatusOK {
		t.Fatalf("Recover failed: %s", wRec.Body.String())
	}

	var recoverResp struct {
		Persona string `json:"persona"`
		ID      string `json:"id"`
		Name    string `json:"name"`
	}
	json.Unmarshal(wRec.Body.Bytes(), &recoverResp)

	if recoverResp.Persona != "client" || recoverResp.ID != clientID || recoverResp.Name != name {
		t.Errorf("Recovered data mismatch: %+v", recoverResp)
	}

	// 3. Recover admin using secret
	adminRecoverBody, _ := json.Marshal(gin.H{"code": "supersecret"})
	reqAdmin, _ := http.NewRequest("POST", "/persona/recover", strings.NewReader(string(adminRecoverBody)))
	wAdmin := httptest.NewRecorder()
	r.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusOK {
		t.Fatalf("Admin recover failed: %s", wAdmin.Body.String())
	}

	var adminRecoverResp struct {
		Persona string `json:"persona"`
	}
	json.Unmarshal(wAdmin.Body.Bytes(), &adminRecoverResp)
	if adminRecoverResp.Persona != "admin" {
		t.Errorf("Expected admin persona, got %s", adminRecoverResp.Persona)
	}
}

func TestDeleteFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dbPath := "./test_delete.db"
	storageDir := "./test_delete_uploads"
	defer os.Remove(dbPath)
	defer os.RemoveAll(storageDir)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer database.Close()

	h := &Handler{
		DB:          database,
		StorageDir:  storageDir,
		AdminSecret: "admin",
	}

	r := gin.New()
	r.POST("/upload", h.UploadFile)
	r.DELETE("/files/:id", h.DeleteFile)

	// 1. Upload a file
	content := "test content"
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test_delete.txt")
	part.Write([]byte(content))
	writer.Close()

	clientID := "owner-1"
	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Client-ID", clientID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var uploadResp struct {
		ID string `json:"id"`
	}
	json.Unmarshal(w.Body.Bytes(), &uploadResp)
	fileID := uploadResp.ID

	// 2. Try to delete as another client (should fail)
	reqDel, _ := http.NewRequest("DELETE", "/files/"+fileID, nil)
	reqDel.Header.Set("X-Client-ID", "other-client")
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)

	if wDel.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for unauthorized deletion, got %d", wDel.Code)
	}

	// 3. Delete as owner (should succeed)
	reqDelOwner, _ := http.NewRequest("DELETE", "/files/"+fileID, nil)
	reqDelOwner.Header.Set("X-Client-ID", clientID)
	wDelOwner := httptest.NewRecorder()
	r.ServeHTTP(wDelOwner, reqDelOwner)

	if wDelOwner.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for owner deletion, got %d: %s", wDelOwner.Code, wDelOwner.Body.String())
	}

	// 4. Verify DB record is gone
	_, err = db.GetFileRecord(database, fileID)
	if err == nil {
		t.Error("Expected DB record to be deleted")
	}

	// 5. Upload another file and delete as admin
	body2 := &bytes.Buffer{}
	writer2 := multipart.NewWriter(body2)
	part2, _ := writer2.CreateFormFile("file", "test_admin_delete.txt")
	part2.Write([]byte(content))
	writer2.Close()

	req2, _ := http.NewRequest("POST", "/upload", body2)
	req2.Header.Set("Content-Type", writer2.FormDataContentType())
	req2.Header.Set("X-Client-ID", clientID)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var uploadResp2 struct {
		ID string `json:"id"`
	}
	json.Unmarshal(w2.Body.Bytes(), &uploadResp2)
	fileID2 := uploadResp2.ID

	reqDelAdmin, _ := http.NewRequest("DELETE", "/files/"+fileID2, nil)
	reqDelAdmin.Header.Set("X-Admin-Secret", "admin")
	wDelAdmin := httptest.NewRecorder()
	r.ServeHTTP(wDelAdmin, reqDelAdmin)

	if wDelAdmin.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for admin deletion, got %d: %s", wDelAdmin.Code, wDelAdmin.Body.String())
	}
}

func TestUploadAndListFiles(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	dbPath := "./test_list_depot.db"
	storageDir := "./test_list_uploads"
	defer os.Remove(dbPath)
	defer os.RemoveAll(storageDir)

	database, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer database.Close()

	h := &Handler{
		DB:         database,
		StorageDir: storageDir,
	}

	r := gin.New()
	r.POST("/upload", h.UploadFile)
	r.GET("/files", h.ListFiles)

	clientID := "test-client-1"

	// 1. Upload a file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test_list.txt")
	io.Copy(part, bytes.NewBufferString("content"))
	writer.Close()

	req, _ := http.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Client-ID", clientID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Upload failed: %s", w.Body.String())
	}

	// 2. List files as client
	reqList, _ := http.NewRequest("GET", "/files", nil)
	reqList.Header.Set("X-Client-ID", clientID)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("List files failed: %s", wList.Body.String())
	}

	var response db.FileListResponse
	json.Unmarshal(wList.Body.Bytes(), &response)
	records := response.Files

	if len(records) == 0 {
		t.Errorf("Expected at least one record for client, got 0")
	}

	// 3. List files as admin
	reqAdmin, _ := http.NewRequest("GET", "/files", nil)
	reqAdmin.RemoteAddr = "127.0.0.1:1234"
	wAdmin := httptest.NewRecorder()
	r.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusOK {
		t.Fatalf("Admin list files failed: %s", wAdmin.Body.String())
	}

	var responseAdmin db.FileListResponse
	json.Unmarshal(wAdmin.Body.Bytes(), &responseAdmin)
	adminRecords := responseAdmin.Files

	if len(adminRecords) == 0 {
		t.Errorf("Expected at least one record for admin, got 0")
	}

	// 4. Test NULL owner_id migration simulation
	// Insert a record with NULL owner_id manually
	_, err = database.Exec("INSERT INTO files (id, original_name, stored_path, size, upload_time, owner_id) VALUES (?, ?, ?, ?, ?, NULL)",
		"null-owner-id", "null_test.txt", "/tmp/null", 100, 1000)
	if err != nil {
		t.Fatalf("Failed to insert NULL owner record: %v", err)
	}

	// List files again, it should not fail now
	wAdmin2 := httptest.NewRecorder()
	r.ServeHTTP(wAdmin2, reqAdmin)
	if wAdmin2.Code != http.StatusOK {
		t.Fatalf("Admin list files with NULL owner failed: %s", wAdmin2.Body.String())
	}

	var responseAdmin2 db.FileListResponse
	json.Unmarshal(wAdmin2.Body.Bytes(), &responseAdmin2)
	adminRecords2 := responseAdmin2.Files
	foundNull := false
	for _, rec := range adminRecords2 {
		if rec.ID == "null-owner-id" {
			foundNull = true
			if rec.OwnerID != "admin" {
				t.Errorf("Expected COALESCE to return 'admin' for NULL owner_id, got %s", rec.OwnerID)
			}
		}
	}
	if !foundNull {
		t.Errorf("Did not find the record that had NULL owner_id")
	}

	// 5. Test with mixed IPs (non-local)
	reqClient2, _ := http.NewRequest("GET", "/files", nil)
	reqClient2.RemoteAddr = "1.2.3.4:1234"
	reqClient2.Header.Set("X-Client-ID", clientID)
	wClient2 := httptest.NewRecorder()
	r.ServeHTTP(wClient2, reqClient2)

	if wClient2.Code != http.StatusOK {
		t.Fatalf("Client 2 list files failed: %s", wClient2.Body.String())
	}
	var responseClient2 db.FileListResponse
	json.Unmarshal(wClient2.Body.Bytes(), &responseClient2)
	client2Records := responseClient2.Files
	if len(client2Records) == 0 {
		t.Errorf("Expected at least one record for client 2, got 0")
	}
}
