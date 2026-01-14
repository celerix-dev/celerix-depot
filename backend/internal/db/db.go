package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type FileRecord struct {
	ID           string `json:"id"`
	OriginalName string `json:"original_name"`
	StoredPath   string `json:"stored_path"`
	Size         int64  `json:"size"`
	UploadTime   int64  `json:"upload_time"`
	OwnerID      string `json:"owner_id"`
	OwnerName    string `json:"owner_name"`
	DownloadLink string `json:"download_link"`
}

type ListFilesOptions struct {
	Search  string
	OwnerID string
	Limit   int
	Offset  int
}

type FileListResponse struct {
	Files []FileRecord `json:"files"`
	Total int          `json:"total"`
}

type ClientRecord struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	RecoveryCode string `json:"recovery_code"`
}

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Create a files table
	query := `
	CREATE TABLE IF NOT EXISTS files (
		id TEXT PRIMARY KEY,
		original_name TEXT,
		stored_path TEXT,
		size INTEGER,
		upload_time INTEGER,
		owner_id TEXT,
		download_link TEXT
	);`

	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	// Create clients table
	query = `
	CREATE TABLE IF NOT EXISTS clients (
		id TEXT PRIMARY KEY,
		name TEXT,
		recovery_code TEXT UNIQUE
	);`

	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	// Migrations
	// Files table migrations
	fileColumns := map[string]string{
		"owner_id":      "TEXT",
		"download_link": "TEXT",
	}

	for col, colType := range fileColumns {
		var count int
		err = db.QueryRow("SELECT count(*) FROM pragma_table_info('files') WHERE name=?", col).Scan(&count)
		if err == nil && count == 0 {
			_, err = db.Exec("ALTER TABLE files ADD COLUMN " + col + " " + colType)
			if err != nil {
				return nil, err
			}
		}
	}

	// Clients table migrations
	clientColumns := map[string]string{
		"recovery_code": "TEXT",
	}

	for col, colType := range clientColumns {
		var count int
		err = db.QueryRow("SELECT count(*) FROM pragma_table_info('clients') WHERE name=?", col).Scan(&count)
		if err == nil && count == 0 {
			_, err = db.Exec("ALTER TABLE clients ADD COLUMN " + col + " " + colType)
			if err != nil {
				return nil, err
			}
		}
	}

	// Ensure no NULL owner_ids exist
	_, err = db.Exec("UPDATE files SET owner_id = 'admin' WHERE owner_id IS NULL")
	if err != nil {
		return nil, err
	}

	return db, nil
}

func SaveFileRecord(db *sql.DB, record FileRecord) error {
	query := `INSERT INTO files (id, original_name, stored_path, size, upload_time, owner_id, download_link) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := db.Exec(query, record.ID, record.OriginalName, record.StoredPath, record.Size, record.UploadTime, record.OwnerID, record.DownloadLink)
	return err
}

func UpdateFileRecord(db *sql.DB, id string, name string, ownerID string) error {
	query := `UPDATE files SET original_name = ?, owner_id = ? WHERE id = ?`
	_, err := db.Exec(query, name, ownerID, id)
	return err
}

func DeleteFileRecord(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM files WHERE id = ?", id)
	return err
}

func GetFileRecord(db *sql.DB, id string) (*FileRecord, error) {
	query := `SELECT f.id, f.original_name, f.stored_path, f.size, f.upload_time, COALESCE(f.owner_id, 'admin'), COALESCE(c.name, 'Unknown'), COALESCE(f.download_link, '') 
	          FROM files f LEFT JOIN clients c ON f.owner_id = c.id WHERE f.id = ?`
	row := db.QueryRow(query, id)

	var record FileRecord
	err := row.Scan(&record.ID, &record.OriginalName, &record.StoredPath, &record.Size, &record.UploadTime, &record.OwnerID, &record.OwnerName, &record.DownloadLink)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func ListFiles(db *sql.DB, opts ListFilesOptions) (*FileListResponse, error) {
	where := "WHERE 1=1"
	args := []interface{}{}

	if opts.OwnerID != "" {
		where += " AND f.owner_id = ?"
		args = append(args, opts.OwnerID)
	}

	if opts.Search != "" {
		where += " AND f.original_name LIKE ?"
		args = append(args, "%"+opts.Search+"%")
	}

	// Count total records
	countQuery := "SELECT COUNT(*) FROM files f " + where
	var total int
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, err
	}

	// Get records
	query := `SELECT f.id, f.original_name, f.stored_path, f.size, f.upload_time, COALESCE(f.owner_id, 'admin'), COALESCE(c.name, 'Unknown'), COALESCE(f.download_link, '') 
	          FROM files f LEFT JOIN clients c ON f.owner_id = c.id ` + where + ` ORDER BY f.upload_time DESC`

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
		if opts.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, opts.Offset)
		}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []FileRecord
	for rows.Next() {
		var record FileRecord
		err := rows.Scan(&record.ID, &record.OriginalName, &record.StoredPath, &record.Size, &record.UploadTime, &record.OwnerID, &record.OwnerName, &record.DownloadLink)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return &FileListResponse{
		Files: records,
		Total: total,
	}, nil
}

func GetAllFileRecords(db *sql.DB) ([]FileRecord, error) {
	resp, err := ListFiles(db, ListFilesOptions{})
	if err != nil {
		return nil, err
	}
	return resp.Files, nil
}

func GetFileRecordsByOwner(db *sql.DB, ownerID string) ([]FileRecord, error) {
	resp, err := ListFiles(db, ListFilesOptions{OwnerID: ownerID})
	if err != nil {
		return nil, err
	}
	return resp.Files, nil
}

func UpsertClient(db *sql.DB, id, name, recoveryCode string) error {
	query := `INSERT INTO clients (id, name, recovery_code) VALUES (?, ?, ?) 
	          ON CONFLICT(id) DO UPDATE SET name=excluded.name`
	_, err := db.Exec(query, id, name, recoveryCode)
	return err
}

func GetClient(db *sql.DB, id string) (*ClientRecord, error) {
	query := `SELECT id, name, COALESCE(recovery_code, '') FROM clients WHERE id = ?`
	row := db.QueryRow(query, id)
	var client ClientRecord
	err := row.Scan(&client.ID, &client.Name, &client.RecoveryCode)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func GetClientByRecoveryCode(db *sql.DB, code string) (*ClientRecord, error) {
	query := `SELECT id, name, recovery_code FROM clients WHERE recovery_code = ?`
	row := db.QueryRow(query, code)
	var client ClientRecord
	err := row.Scan(&client.ID, &client.Name, &client.RecoveryCode)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func ListClients(db *sql.DB) ([]ClientRecord, error) {
	query := `SELECT id, name, COALESCE(recovery_code, '') FROM clients ORDER BY name ASC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []ClientRecord
	for rows.Next() {
		var c ClientRecord
		if err := rows.Scan(&c.ID, &c.Name, &c.RecoveryCode); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, nil
}

func UpdateClient(db *sql.DB, id string, name string, recoveryCode string) error {
	query := `UPDATE clients SET name = ?, recovery_code = ? WHERE id = ?`
	_, err := db.Exec(query, name, recoveryCode, id)
	return err
}
