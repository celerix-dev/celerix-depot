package db

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type CelerixStore interface {
	Get(personaID, appID, key string) (any, error)
	Set(personaID, appID, key string, val any) error
	Delete(personaID, appID, key string) error
	GetApps(personaID string) ([]string, error)
	GetPersonas() ([]string, error)
	GetAppStore(personaID, appID string) (map[string]any, error)
	DumpApp(appID string) (map[string]map[string]any, error)
	GetGlobal(appID, key string) (any, string, error)
	Move(srcPersona, dstPersona, appID, key string) error
}

func getRecord[T any](s CelerixStore, personaID, appID, key string) (T, error) {
	var target T
	val, err := s.Get(personaID, appID, key)
	if err != nil {
		return target, err
	}

	if v, ok := val.(T); ok {
		return v, nil
	}

	bytes, err := json.Marshal(val)
	if err != nil {
		return target, err
	}
	err = json.Unmarshal(bytes, &target)
	return target, err
}

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
	LastActive   int64  `json:"last_active"`
	IsAdmin      bool   `json:"is_admin"`
}

const (
	AppID           = "depot"
	FileKeyPrefix   = "file:"
	ClientKeyPrefix = "client:"
	SystemPersona   = "_system"
)

func SaveFileRecord(s CelerixStore, record FileRecord) error {
	persona := record.OwnerID
	if persona == "" {
		persona = SystemPersona
	}
	return s.Set(persona, AppID, FileKeyPrefix+record.ID, record)
}

func UpdateFileRecord(s CelerixStore, id string, name string, ownerID string) error {
	record, err := GetFileRecord(s, id)
	if err != nil {
		return err
	}
	oldPersona := record.OwnerID
	if oldPersona == "" {
		oldPersona = SystemPersona
	}

	record.OriginalName = name
	record.OwnerID = ownerID

	newPersona := ownerID
	if newPersona == "" {
		newPersona = SystemPersona
	}

	if oldPersona != newPersona {
		if err := s.Move(oldPersona, newPersona, AppID, FileKeyPrefix+id); err != nil {
			return err
		}
	}

	// Always update the record content
	return s.Set(newPersona, AppID, FileKeyPrefix+record.ID, record)
}

func DeleteFileRecord(s CelerixStore, id string) error {
	record, err := GetFileRecord(s, id)
	if err != nil {
		return err
	}
	persona := record.OwnerID
	if persona == "" {
		persona = SystemPersona
	}
	return s.Delete(persona, AppID, FileKeyPrefix+id)
}

func GetFileRecord(s CelerixStore, id string) (*FileRecord, error) {
	_, personaID, err := s.GetGlobal(AppID, FileKeyPrefix+id)
	if err != nil {
		return nil, err
	}

	record, err := getRecord[FileRecord](s, personaID, AppID, FileKeyPrefix+id)
	if err != nil {
		return nil, err
	}

	// Fetch owner name
	if record.OwnerID != "" {
		client, err := GetClient(s, record.OwnerID)
		if err == nil {
			record.OwnerName = client.Name
		} else {
			record.OwnerName = "Unknown"
		}
	} else {
		record.OwnerName = "Admin"
	}

	return &record, nil
}

func ListFiles(s CelerixStore, opts ListFilesOptions) (*FileListResponse, error) {
	var allRecords []FileRecord

	if opts.OwnerID != "" {
		// Optimization: if we have OwnerID, only look in that persona
		appStore, err := s.GetAppStore(opts.OwnerID, AppID)
		if err == nil {
			for k := range appStore {
				if strings.HasPrefix(k, FileKeyPrefix) {
					// Using sdk.Get for individual item to ensure type safety if needed
					r, err := getRecord[FileRecord](s, opts.OwnerID, AppID, k)
					if err == nil {
						allRecords = append(allRecords, r)
					}
				}
			}
		}
	} else {
		// Admin view or no owner specified: use DumpApp for efficiency
		allData, err := s.DumpApp(AppID)
		if err != nil {
			return nil, err
		}

		for personaID, appStore := range allData {
			for k := range appStore {
				if strings.HasPrefix(k, FileKeyPrefix) {
					r, err := getRecord[FileRecord](s, personaID, AppID, k)
					if err == nil {
						allRecords = append(allRecords, r)
					}
				}
			}
		}
	}

	var filtered []FileRecord
	for _, r := range allRecords {
		// Filter by search
		if opts.Search != "" && !strings.Contains(strings.ToLower(r.OriginalName), strings.ToLower(opts.Search)) {
			continue
		}

		// Fetch owner name
		if r.OwnerID != "" {
			client, err := GetClient(s, r.OwnerID)
			if err == nil {
				r.OwnerName = client.Name
			} else {
				r.OwnerName = "Unknown"
			}
		} else {
			r.OwnerName = "Admin"
		}
		filtered = append(filtered, r)
	}

	// Sort by upload time desc
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].UploadTime > filtered[j].UploadTime
	})

	total := len(filtered)

	// Pagination
	start := opts.Offset
	if start > total {
		start = total
	}
	end := start + opts.Limit
	if opts.Limit <= 0 || end > total {
		end = total
	}

	return &FileListResponse{
		Files: filtered[start:end],
		Total: total,
	}, nil
}

func GetAllFileRecords(s CelerixStore) ([]FileRecord, error) {
	resp, err := ListFiles(s, ListFilesOptions{})
	if err != nil {
		return nil, err
	}
	return resp.Files, nil
}

func GetFileRecordsByOwner(s CelerixStore, ownerID string) ([]FileRecord, error) {
	resp, err := ListFiles(s, ListFilesOptions{OwnerID: ownerID})
	if err != nil {
		return nil, err
	}
	return resp.Files, nil
}

func UpsertClient(s CelerixStore, id, name, recoveryCode string, lastActive int64) error {
	client, err := GetClient(s, id)
	if err != nil {
		// New client
		client = &ClientRecord{
			ID:           id,
			Name:         name,
			RecoveryCode: recoveryCode,
			LastActive:   lastActive,
			IsAdmin:      false,
		}
	} else {
		client.Name = name
		client.RecoveryCode = recoveryCode
		client.LastActive = lastActive
	}
	return s.Set(SystemPersona, AppID, ClientKeyPrefix+id, client)
}

func UpdateClientLastActive(s CelerixStore, id string, lastActive int64) error {
	client, err := GetClient(s, id)
	if err != nil {
		return err
	}
	client.LastActive = lastActive
	return s.Set(SystemPersona, AppID, ClientKeyPrefix+id, client)
}

func DeleteClient(s CelerixStore, id string) error {
	return s.Delete(SystemPersona, AppID, ClientKeyPrefix+id)
}

func GetClient(s CelerixStore, id string) (*ClientRecord, error) {
	client, err := getRecord[ClientRecord](s, SystemPersona, AppID, ClientKeyPrefix+id)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func GetClientByRecoveryCode(s CelerixStore, code string) (*ClientRecord, error) {
	appStore, err := s.GetAppStore(SystemPersona, AppID)
	if err != nil {
		return nil, err
	}

	for k := range appStore {
		if strings.HasPrefix(k, ClientKeyPrefix) {
			c, err := getRecord[ClientRecord](s, SystemPersona, AppID, k)
			if err == nil && c.RecoveryCode == code {
				return &c, nil
			}
		}
	}
	return nil, fmt.Errorf("client not found")
}

func ListClients(s CelerixStore) ([]ClientRecord, error) {
	appStore, err := s.GetAppStore(SystemPersona, AppID)
	if err != nil {
		return nil, err
	}

	var clients []ClientRecord
	for k := range appStore {
		if strings.HasPrefix(k, ClientKeyPrefix) {
			c, err := getRecord[ClientRecord](s, SystemPersona, AppID, k)
			if err == nil {
				clients = append(clients, c)
			}
		}
	}

	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Name < clients[j].Name
	})

	return clients, nil
}

func UpdateClientAdminStatus(s CelerixStore, id string, isAdmin bool) error {
	client, err := GetClient(s, id)
	if err != nil {
		return err
	}
	client.IsAdmin = isAdmin
	return s.Set(SystemPersona, AppID, ClientKeyPrefix+id, client)
}

func UpdateClientFull(s CelerixStore, id string, name string, recoveryCode string, isAdmin bool) error {
	client, err := GetClient(s, id)
	if err != nil {
		return err
	}
	client.Name = name
	client.RecoveryCode = recoveryCode
	client.IsAdmin = isAdmin
	return s.Set(SystemPersona, AppID, ClientKeyPrefix+id, client)
}
