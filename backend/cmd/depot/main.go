package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/celerix-dev/celerix-store/pkg/sdk"
	"github.com/celerix/depot/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

//go:embed all:dist
var frontendDist embed.FS

//go:embed version.json
var versionFile []byte

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	storageDir := os.Getenv("STORAGE_DIR")
	if storageDir == "" {
		storageDir = filepath.Join(dataDir, "uploads")
	}

	// Ensure directories exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	namespaceStr := os.Getenv("CELERIX_NAMESPACE")
	if namespaceStr == "" {
		log.Fatal("CELERIX_NAMESPACE environment variable is required")
	}
	celerixNamespace, err := uuid.Parse(namespaceStr)
	if err != nil {
		log.Fatalf("Failed to parse CELERIX_NAMESPACE as UUID: %v", err)
	}

	store, err := sdk.New(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize Celerix Store: %v", err)
	}

	h := &api.Handler{
		Store:            store,
		StorageDir:       storageDir,
		AdminSecret:      os.Getenv("ADMIN_SECRET"),
		VersionConfig:    versionFile,
		CelerixNamespace: celerixNamespace,
	}

	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-Client-ID, X-Admin-Secret")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/version", h.GetVersion)
		apiGroup.GET("/persona", h.GetPersona)
		apiGroup.POST("/persona/name", h.UpdateClientName)
		apiGroup.POST("/persona/recover", h.RecoverPersona)
		apiGroup.POST("/persona/admin", h.ActivateAdmin)
		apiGroup.POST("/upload", h.UploadFile)
		apiGroup.GET("/files", h.ListFiles)
		apiGroup.GET("/files/:id", h.GetFileMetadata)
		apiGroup.PUT("/files/:id", h.UpdateFile)
		apiGroup.DELETE("/files/:id", h.DeleteFile)
		apiGroup.GET("/clients", h.ListClients)
		apiGroup.PUT("/clients/:id", h.UpdateClient)
		apiGroup.DELETE("/clients/:id", h.DeleteClient)
		apiGroup.GET("/download/:id", h.DownloadFile)
	}

	// Serve frontend static files
	distFS, err := fs.Sub(frontendDist, "dist")
	if err != nil {
		log.Fatalf("Failed to sub embedded dist: %v", err)
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// If it's an API request that reached here, return 404
		if strings.HasPrefix(path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}

		// Try to serve the file from the embedded filesystem
		file, err := distFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			file.Close()
			http.FileServer(http.FS(distFS)).ServeHTTP(c.Writer, c.Request)
			return
		}

		// Fallback to index.html for SPA routing
		c.FileFromFS("/", http.FS(distFS))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
