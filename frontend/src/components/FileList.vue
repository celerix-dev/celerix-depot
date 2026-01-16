<script setup lang="ts">
import { onMounted, ref, computed } from 'vue';
import dayjs from 'dayjs';
import { getClientID, getAdminSecret } from '@/utils/persona';

const props = defineProps<{
  persona: string
}>();

interface FileRecord {
  id: string;
  original_name: string;
  size: number;
  upload_time: number;
  owner_id: string;
  owner_name: string;
  download_link: string;
  is_public: boolean;
}

const files = ref<FileRecord[]>([]);
const total = ref(0);
const search = ref('');
const currentPage = ref(1);
const limit = 8;
const currentClientID = getClientID();

const fetchFiles = async () => {
  console.log('Fetching files for client:', getClientID());
  try {
    const params = new URLSearchParams({
      page: currentPage.value.toString(),
      limit: limit.toString(),
      search: search.value
    });
    
    const response = await fetch(`/api/files?${params.toString()}`, {
      headers: {
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
    });
    if (response.ok) {
      const data = await response.json();
      files.value = data.files;
      total.value = data.total;
      console.log('Fetched files:', files.value, 'Total:', total.value);
    } else {
      console.error('Failed to fetch files:', response.status, response.statusText);
    }
  } catch (error) {
    console.error('Error fetching files:', error);
  }
};

const totalPages = computed(() => Math.ceil(total.value / limit));

const onSearch = () => {
  currentPage.value = 1;
  fetchFiles();
};

const changePage = (page: number) => {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page;
    fetchFiles();
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

const getDownloadUrl = (record: FileRecord) => {
  // If we have a public download link, use it. Otherwise fallback to ID.
  const link = record.download_link || record.id;
  return `/api/download/${link}`;
};

const copyToClipboard = (text: string) => {
  navigator.clipboard.writeText(text).then(() => {
    alert('Link copied to clipboard!');
  });
};

const deleteFile = async (file: FileRecord) => {
  if (!confirm(`Are you sure you want to delete "${file.original_name}"?`)) {
    return;
  }

  try {
    const response = await fetch(`/api/files/${file.id}`, {
      method: 'DELETE',
      headers: {
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
    });

    if (response.ok) {
      await fetchFiles();
    } else {
      const data = await response.json().catch(() => ({}));
      alert(`Failed to delete file: ${data.error || response.statusText}`);
    }
  } catch (error) {
    console.error('Error deleting file:', error);
    alert('Error deleting file.');
  }
};

const togglePublic = async (file: FileRecord) => {
  try {
    const response = await fetch(`/api/files/${file.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
      body: JSON.stringify({
        original_name: file.original_name,
        owner_id: file.owner_id,
        is_public: !file.is_public
      }),
    });

    if (response.ok) {
      await fetchFiles();
    } else {
      const data = await response.json().catch(() => ({}));
      alert(`Failed to update file visibility: ${data.error || response.statusText}`);
    }
  } catch (error) {
    console.error('Error updating file visibility:', error);
    alert('Error updating file visibility.');
  }
};

onMounted(fetchFiles);

defineExpose({ fetchFiles });
</script>

<template>
  <div class="card shadow-sm">
    <div class="card-body">
      <div class="d-flex justify-content-between align-items-center mb-4">
        <h5 class="card-title mb-0">Files</h5>
        <div class="input-group w-50">
          <span class="input-group-text bg-transparent border-end-0">
            <i class="ti ti-search text-muted"></i>
          </span>
          <input 
            v-model="search" 
            type="text" 
            class="form-control border-start-0" 
            placeholder="Search files..." 
            @input="onSearch"
          />
        </div>
      </div>

      <div v-if="files.length === 0" class="text-center text-muted p-4">
        {{ search ? 'No files match your search.' : 'No files uploaded yet.' }}
      </div>
      <div v-else>
        <div class="table-responsive">
          <table class="table table-hover align-middle">
            <thead>
              <tr>
                <th>Name</th>
                <th v-if="persona === 'admin'">Owner</th>
                <th v-else>Owner</th>
                <th>Size</th>
                <th>Uploaded At</th>
                <th class="text-end">Action</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="file in files" :key="file.id">
                <td>
                  <i class="ti ti-file me-2"></i>
                  {{ file.original_name }}
                  <div class="d-inline-block ms-2">
                    <span v-if="file.is_public" class="badge bg-info-subtle text-info border">
                      Public
                    </span>
                    <button v-if="file.owner_id === currentClientID || persona === 'admin'" 
                            class="btn btn-xs ms-1 p-0 border-0" 
                            @click="togglePublic(file)"
                            :title="file.is_public ? 'Make Private' : 'Make Public'">
                      <i :class="['ti', file.is_public ? 'ti-lock-open text-info' : 'ti-lock text-muted']"></i>
                    </button>
                  </div>
                </td>
                <td v-if="persona === 'admin'">
                  <span class="badge bg-secondary-subtle text-secondary border">
                    {{ file.owner_name }}
                  </span>
                </td>
                <td v-else>
                  <span class="badge bg-primary-subtle text-primary border">
                    {{ file.owner_id === currentClientID ? 'You' : (file.owner_name || 'Unknown') }}
                  </span>
                </td>
                <td>{{ formatSize(file.size) }}</td>
                <td>{{ formatDate(file.upload_time) }}</td>
                <td class="text-end">
                  <div class="btn-group">
                    <a :href="getDownloadUrl(file)" class="btn btn-sm btn-outline-primary" download>
                      <i class="ti ti-download me-1"></i>
                      Download
                    </a>
                    <button class="btn btn-sm btn-outline-secondary" @click="copyToClipboard(getDownloadUrl(file))">
                      <i class="ti ti-copy me-1"></i>
                      Link
                    </button>
                    <button class="btn btn-sm btn-outline-danger" @click="deleteFile(file)">
                      <i class="ti ti-trash"></i>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Pagination -->
        <div v-if="totalPages > 1" class="d-flex justify-content-between align-items-center mt-3">
          <span class="text-muted small">
            Showing {{ (currentPage - 1) * limit + 1 }} to {{ Math.min(currentPage * limit, total) }} of {{ total }} files
          </span>
          <nav>
            <ul class="pagination pagination-sm mb-0">
              <li class="page-item" :class="{ disabled: currentPage === 1 }">
                <button class="page-item page-link" @click="changePage(currentPage - 1)">
                  <i class="ti ti-chevron-left"></i>
                </button>
              </li>
              <li 
                v-for="page in totalPages" 
                :key="page" 
                class="page-item" 
                :class="{ active: currentPage === page }"
              >
                <button class="page-item page-link" @click="changePage(page)">{{ page }}</button>
              </li>
              <li class="page-item" :class="{ disabled: currentPage === totalPages }">
                <button class="page-item page-link" @click="changePage(currentPage + 1)">
                  <i class="ti ti-chevron-right"></i>
                </button>
              </li>
            </ul>
          </nav>
        </div>
      </div>
    </div>
  </div>
</template>
