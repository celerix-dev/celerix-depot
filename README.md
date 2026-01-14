# Celerix Depot

Celerix Depot is a lightweight, self-hosted file sharing service designed for "homelab" use. It allows users to easily upload, store, and share files through a clean, persona-based web interface.

## 🚀 Features

- **Drag-and-Drop Uploads**: Simple and intuitive interface for uploading files.
- **Persona System**: 
  - **Admin Persona**: Full visibility and management of all uploaded files and user personas.
  - **Client Persona**: Users see and manage only their own uploads.
- **Privacy & Public Sharing**: Files are private to the owner by default, but each upload generates a unique, non-guessable public download link for easy sharing.
- **Persona Recovery**: Clients can restore their identity across devices using a short, 8-character recovery code.
- **Admin Management**: Dedicated interface for administrators to edit file metadata and manage client personas (names and recovery codes).
- **Filtering & Pagination**: Efficiently browse large file collections with real-time search and 8-item-per-page pagination.
- **Docker Ready**: Fully containerized for easy deployment on NAS or home servers.
- **Modern UI**: Built with Vue 3, featuring a responsive design with dark mode support.

## 🛠 Tech Stack

### Backend
- **Language**: Go
- **Framework**: Gin Gonic (Web)
- **Database**: SQLite (Metadata indexing)
- **Storage**: Local Disk (Streamed uploads)

### Frontend
- **Framework**: Vue 3 (Composition API)
- **Build Tool**: Vite
- **Styling**: Bootstrap / Halfmoon / Tabler Icons

## 📦 Deployment (Docker Compose)

The easiest way to run Celerix Depot is using Docker Compose.

1. Create a `docker-compose.yml` file:

```yaml
services:
  depot:
    build: .
    container_name: celerix-depot
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      - ADMIN_SECRET=your-secret-key-here
    restart: unless-stopped
```

2. Run the service:
```bash
docker-compose up -d
```

Your Depot is now reachable at `http://your-server-ip:8080`.

## ⚙️ Configuration

Celerix Depot can be configured using the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | The port the service listens on inside the container. | `8080` |
| `DB_PATH` | Path to the SQLite database file. | `/app/data/depot.db` |
| `STORAGE_DIR`| Directory where uploaded files are stored. | `/app/data/uploads` |
| `ADMIN_SECRET`| The secret key used to activate the Admin Persona. | `admin123` |

## 🔑 Personas & Recovery

- **Becoming Admin**: Click the settings (gear) icon in the header and enter the `ADMIN_SECRET`.
- **Client Identification**: Every browser is assigned a unique `X-Client-ID` stored in `localStorage`.
- **Identity Recovery**: 
  - When setting a name, clients are given a **Recovery Code**. 
  - If `localStorage` is cleared, users can click the "Recover Persona" (refresh) icon and enter their code to regain access to their files.
  - Admins can view and regenerate recovery codes via the Admin Management panel.

## 👨‍💻 Development

### Backend
```bash
cd backend
go mod download
go run cmd/depot/main.go
```

### Frontend
```bash
cd frontend
npm install
npm run dev
```

### Build from Source
```bash
docker build -t celerix-depot .
```

## 📄 License
MIT
