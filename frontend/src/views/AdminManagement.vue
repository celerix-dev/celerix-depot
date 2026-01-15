<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { getAdminSecret, getClientID, fetchPersona } from '@/utils/persona';
import dayjs from 'dayjs';

interface FileRecord {
  id: string;
  original_name: string;
  size: number;
  upload_time: number;
  owner_id: string;
  owner_name: string;
  download_link: string;
}

interface ClientRecord {
  id: string;
  name: string;
  recovery_code: string;
  last_active: number;
}

const activeTab = ref<'files' | 'clients'>('files');
const appVersion = ref('');

// Files management
const files = ref<FileRecord[]>([]);
const editingFileId = ref<string | null>(null);
const fileEditForm = ref({
  original_name: '',
  owner_id: ''
});

const fetchAllFiles = async () => {
  try {
    const response = await fetch('/api/files?limit=100', {
      headers: {
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
    });
    if (response.ok) {
      const data = await response.json();
      files.value = data.files;
    }
  } catch (error) {
    console.error('Error fetching files:', error);
  }
};

const startEditFile = (file: FileRecord) => {
  editingFileId.value = file.id;
  fileEditForm.value = {
    original_name: file.original_name,
    owner_id: file.owner_id
  };
};

const cancelEditFile = () => {
  editingFileId.value = null;
};

const saveEditFile = async (id: string) => {
  try {
    const response = await fetch(`/api/files/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
      body: JSON.stringify(fileEditForm.value),
    });

    if (response.ok) {
      editingFileId.value = null;
      await fetchAllFiles();
    } else {
      alert('Failed to update file');
    }
  } catch (error) {
    console.error('Error updating file:', error);
    alert('Error updating file');
  }
};

// Clients management
const clients = ref<ClientRecord[]>([]);
const editingClientId = ref<string | null>(null);
const clientEditForm = ref({
  name: '',
  recovery_code: ''
});

const fetchAllClients = async () => {
  try {
    const response = await fetch('/api/clients', {
      headers: {
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
    });
    if (response.ok) {
      clients.value = await response.json();
    }
  } catch (error) {
    console.error('Error fetching clients:', error);
  }
};

const startEditClient = (client: ClientRecord) => {
  editingClientId.value = client.id;
  clientEditForm.value = {
    name: client.name,
    recovery_code: client.recovery_code
  };
};

const cancelEditClient = () => {
  editingClientId.value = null;
};

const saveEditClient = async (id: string) => {
  try {
    const response = await fetch(`/api/clients/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
      body: JSON.stringify(clientEditForm.value),
    });

    if (response.ok) {
      editingClientId.value = null;
      await fetchAllClients();
    } else {
      alert('Failed to update client');
    }
  } catch (error) {
    console.error('Error updating client:', error);
    alert('Error updating client');
  }
};

const deleteClient = async (id: string, name: string) => {
  if (!confirm(`Are you sure you want to delete client "${name}"? This will NOT delete their files, but they will be listed as "Unknown".`)) {
    return;
  }

  try {
    const response = await fetch(`/api/clients/${id}`, {
      method: 'DELETE',
      headers: {
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
    });

    if (response.ok) {
      await fetchAllClients();
    } else {
      const errorData = await response.json().catch(() => ({}));
      alert(`Failed to delete client: ${errorData.error || response.statusText}`);
    }
  } catch (error) {
    console.error('Error deleting client:', error);
    alert('Error deleting client');
  }
};

const deleteFile = async (id: string, name: string) => {
  if (!confirm(`Are you sure you want to delete "${name}"?`)) {
    return;
  }

  try {
    const response = await fetch(`/api/files/${id}`, {
      method: 'DELETE',
      headers: {
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
    });

    if (response.ok) {
      await fetchAllFiles();
    } else {
      alert('Failed to delete file');
    }
  } catch (error) {
    console.error('Error deleting file:', error);
    alert('Error deleting file');
  }
};

const formatSize = (bytes: number) => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const formatDate = (timestamp: number) => {
  return dayjs(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss');
};

onMounted(async () => {
  const data = await fetchPersona();
  appVersion.value = data.version || '';
  fetchAllFiles();
  fetchAllClients();
});
</script>

<template>
  <div class="container py-5">
    <div class="d-flex justify-content-between align-items-center mb-4">
      <div class="d-flex align-items-center">
        <h2 class="mb-0 me-3">Admin Management</h2>
        <div v-if="appVersion"><small class="text-muted">v{{ appVersion }}</small></div>
      </div>
      <router-link to="/" class="btn btn-outline-secondary">Back to Gallery</router-link>
    </div>

    <ul class="nav nav-tabs mb-4">
      <li class="nav-item">
        <button 
          class="nav-link" 
          :class="{ active: activeTab === 'files' }" 
          @click="activeTab = 'files'"
        >
          Files
        </button>
      </li>
      <li class="nav-item">
        <button 
          class="nav-link" 
          :class="{ active: activeTab === 'clients' }" 
          @click="activeTab = 'clients'"
        >
          Clients (Personas)
        </button>
      </li>
    </ul>

    <div class="card shadow-sm">
      <div class="card-body">
        <!-- Files Table -->
        <div v-if="activeTab === 'files'" class="table-responsive">
          <table class="table table-hover align-middle">
            <thead>
              <tr>
                <th>ID</th>
                <th>Filename</th>
                <th>Owner ID</th>
                <th>Owner Name</th>
                <th>Size</th>
                <th>Upload Time</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="file in files" :key="file.id">
                <td><small class="text-muted">{{ file.id.substring(0, 8) }}...</small></td>
                <td>
                  <template v-if="editingFileId === file.id">
                    <input v-model="fileEditForm.original_name" type="text" class="form-control form-control-sm" />
                  </template>
                  <template v-else>
                    {{ file.original_name }}
                  </template>
                </td>
                <td>
                  <template v-if="editingFileId === file.id">
                    <input v-model="fileEditForm.owner_id" type="text" class="form-control form-control-sm" />
                  </template>
                  <template v-else>
                    <small>{{ file.owner_id }}</small>
                  </template>
                </td>
                <td>{{ file.owner_name }}</td>
                <td>{{ formatSize(file.size) }}</td>
                <td>{{ formatDate(file.upload_time) }}</td>
                <td>
                  <div v-if="editingFileId === file.id" class="btn-group btn-group-sm">
                    <button class="btn btn-success" @click="saveEditFile(file.id)">Save</button>
                    <button class="btn btn-secondary" @click="cancelEditFile">Cancel</button>
                  </div>
                  <div v-else class="btn-group btn-group-sm">
                    <button class="btn btn-outline-primary" @click="startEditFile(file)">Edit</button>
                    <button class="btn btn-outline-danger" @click="deleteFile(file.id, file.original_name)">Delete</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Clients Table -->
        <div v-if="activeTab === 'clients'" class="table-responsive">
          <table class="table table-hover align-middle">
            <thead>
              <tr>
                <th>UUID</th>
                <th>Name</th>
                <th>Recovery Code</th>
                <th>Last Active</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="client in clients" :key="client.id">
                <td><small class="text-muted">{{ client.id }}</small></td>
                <td>
                  <template v-if="editingClientId === client.id">
                    <input v-model="clientEditForm.name" type="text" class="form-control form-control-sm" />
                  </template>
                  <template v-else>
                    {{ client.name }}
                  </template>
                </td>
                <td>
                  <template v-if="editingClientId === client.id">
                    <input v-model="clientEditForm.recovery_code" type="text" class="form-control form-control-sm" />
                  </template>
                  <template v-else>
                    <code>{{ client.recovery_code }}</code>
                  </template>
                </td>
                <td>
                  {{ client.last_active ? formatDate(client.last_active) : 'Never' }}
                </td>
                <td>
                  <div v-if="editingClientId === client.id" class="btn-group btn-group-sm">
                    <button class="btn btn-success" @click="saveEditClient(client.id)">Save</button>
                    <button class="btn btn-secondary" @click="cancelEditClient">Cancel</button>
                  </div>
                  <div v-else class="btn-group btn-group-sm">
                    <button class="btn btn-outline-primary" @click="startEditClient(client)">Edit</button>
                    <button class="btn btn-outline-danger" @click="deleteClient(client.id, client.name)">Delete</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>
