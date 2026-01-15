package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/celerix/depot/internal/db"
	"github.com/celerix/depot/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	DB            *sql.DB
	StorageDir    string
	AdminSecret   string
	VersionConfig []byte
}

func (h *Handler) GetVersion(c *gin.Context) {
	c.Data(http.StatusOK, "application/json", h.VersionConfig)
}

func (h *Handler) isAdmin(c *gin.Context) bool {
	// First, check the secret header
	secret := c.GetHeader("X-Admin-Secret")
	if h.AdminSecret != "" && secret == h.AdminSecret {
		return true
	}
	// Fallback to localhost for convenience during development,
	// but the secret is the primary way now.
	ip := c.ClientIP()
	return ip == "127.0.0.1" || ip == "::1"
}

func (h *Handler) GetPersona(c *gin.Context) {
	ownerID := c.GetHeader("X-Client-ID")
	persona := "client"
	if h.isAdmin(c) {
		persona = "admin"
	}

	name := ""
	recoveryCode := ""
	if ownerID != "" {
		client, err := db.GetClient(h.DB, ownerID)
		if err == nil {
			name = client.Name
			recoveryCode = client.RecoveryCode
			// Update last active time
			_ = db.UpdateClientLastActive(h.DB, ownerID, time.Now().Unix())
		}
	} else if persona == "admin" {
		name = "Administrator"
	}

	// Extract version from VersionConfig bytes
	version := "unknown"
	var vCfg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(h.VersionConfig, &vCfg); err == nil {
		version = vCfg.Version
	}

	c.JSON(http.StatusOK, gin.H{
		"persona":       persona,
		"name":          name,
		"recovery_code": recoveryCode,
		"version":       version,
	})
}

func (h *Handler) RecoverPersona(c *gin.Context) {
	var input struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if it's the admin secret
	if h.AdminSecret != "" && input.Code == h.AdminSecret {
		c.JSON(http.StatusOK, gin.H{
			"persona": "admin",
			"id":      "admin",
			"name":    "Administrator",
		})
		return
	}

	// Otherwise, check client recovery codes
	client, err := db.GetClientByRecoveryCode(h.DB, input.Code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid recovery code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"persona": "client",
		"id":      client.ID,
		"name":    client.Name,
	})
}

func (h *Handler) UpdateClientName(c *gin.Context) {
	ownerID := c.GetHeader("X-Client-ID")
	if ownerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Client-ID header is required"})
		return
	}

	var input struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate a recovery code if it's a new client or they don't have one
	client, err := db.GetClient(h.DB, ownerID)
	recoveryCode := ""
	if err == nil && client.RecoveryCode != "" {
		recoveryCode = client.RecoveryCode
	} else {
		// Generate a simple short code
		recoveryCode = strings.ToUpper(uuid.New().String()[:8])
	}

	err = db.UpsertClient(h.DB, ownerID, input.Name, recoveryCode, time.Now().Unix())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update client name"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"recovery_code": recoveryCode,
	})
}

func (h *Handler) UploadFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file is received"})
		return
	}
	defer file.Close()

	ownerID := c.GetHeader("X-Client-ID")
	if ownerID == "" && !h.isAdmin(c) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Client-ID header is required for clients"})
		return
	}
	if h.isAdmin(c) && ownerID == "" {
		ownerID = "admin"
	}

	id := uuid.New().String()
	storedName := id // We use the UUID as the filename on disk for safety

	storedPath, size, err := storage.StoreFile(file, h.StorageDir, storedName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store file: " + err.Error()})
		return
	}

	// Generate public download link
	downloadLink := uuid.New().String()

	record := db.FileRecord{
		ID:           id,
		OriginalName: header.Filename,
		StoredPath:   storedPath,
		Size:         size,
		UploadTime:   time.Now().Unix(),
		OwnerID:      ownerID,
		DownloadLink: downloadLink,
	}

	log.Printf("[DEBUG] Saving record: ID=%s, Name=%s, OwnerID=%s", record.ID, record.OriginalName, record.OwnerID)
	err = db.SaveFileRecord(h.DB, record)
	if err != nil {
		log.Printf("[DEBUG] Failed to save record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save record: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *Handler) ListFiles(c *gin.Context) {
	isAdmin := h.isAdmin(c)
	ownerID := c.GetHeader("X-Client-ID")
	search := c.Query("search")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "8")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 8
	}
	offset := (page - 1) * limit

	opts := db.ListFilesOptions{
		Search: search,
		Limit:  limit,
		Offset: offset,
	}

	if !isAdmin {
		if ownerID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Client-ID header is required"})
			return
		}
		opts.OwnerID = ownerID
	}

	log.Printf("[DEBUG] ListFiles request: isAdmin=%v, X-Client-ID=%s, Search=%s, Page=%d, Limit=%d", isAdmin, ownerID, search, page, limit)

	response, err := db.ListFiles(h.DB, opts)
	if err != nil {
		log.Printf("[DEBUG] Failed to list files: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files"})
		return
	}

	log.Printf("[DEBUG] Returning %d records (Total: %d)", len(response.Files), response.Total)
	c.JSON(http.StatusOK, response)
}

func (h *Handler) DownloadFile(c *gin.Context) {
	idOrLink := c.Param("id")
	// Try finding by ID first
	record, err := db.GetFileRecord(h.DB, idOrLink)
	if err != nil {
		// Try finding by download_link
		query := `SELECT f.id, f.original_name, f.stored_path, f.size, f.upload_time, COALESCE(f.owner_id, 'admin'), COALESCE(c.name, 'Unknown'), COALESCE(f.download_link, '') 
		          FROM files f LEFT JOIN clients c ON f.owner_id = c.id WHERE f.download_link = ?`
		row := h.DB.QueryRow(query, idOrLink)
		record = &db.FileRecord{}
		err = row.Scan(&record.ID, &record.OriginalName, &record.StoredPath, &record.Size, &record.UploadTime, &record.OwnerID, &record.OwnerName, &record.DownloadLink)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}
	}

	c.FileAttachment(record.StoredPath, record.OriginalName)
}

func (h *Handler) GetFileMetadata(c *gin.Context) {
	id := c.Param("id")
	record, err := db.GetFileRecord(h.DB, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.JSON(http.StatusOK, record)
}

func (h *Handler) UpdateFile(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	id := c.Param("id")
	var input struct {
		OriginalName string `json:"original_name" binding:"required"`
		OwnerID      string `json:"owner_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := db.UpdateFileRecord(h.DB, id, input.OriginalName, input.OwnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) DeleteFile(c *gin.Context) {
	id := c.Param("id")
	record, err := db.GetFileRecord(h.DB, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Permission check: admin or owner
	ownerID := c.GetHeader("X-Client-ID")
	if !h.isAdmin(c) && record.OwnerID != ownerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this file"})
		return
	}

	// Delete from storage
	err = storage.DeleteFile(record.StoredPath)
	if err != nil {
		log.Printf("[ERROR] Failed to delete file from storage: %v", err)
		// We continue even if file is missing from storage to clean up DB
	}

	// Delete from DB
	err = db.DeleteFileRecord(h.DB, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) ListClients(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	clients, err := db.ListClients(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list clients"})
		return
	}

	c.JSON(http.StatusOK, clients)
}

func (h *Handler) UpdateClient(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	id := c.Param("id")
	var input struct {
		Name         string `json:"name" binding:"required"`
		RecoveryCode string `json:"recovery_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := db.UpdateClient(h.DB, id, input.Name, input.RecoveryCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update client"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) DeleteClient(c *gin.Context) {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	id := c.Param("id")
	if id == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete admin persona"})
		return
	}

	err := db.DeleteClient(h.DB, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete client"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
