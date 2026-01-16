<script setup lang="ts">
import { ref } from 'vue';
import { getClientID, getAdminSecret } from '@/utils/persona';

const emit = defineEmits(['uploaded']);
const isDragging = ref(false);
const uploadProgress = ref(0);
const isUploading = ref(false);
const uploadStatus = ref('');

const onDrop = (e: DragEvent) => {
  isDragging.value = false;
  if (isUploading.value) return;
  const files = e.dataTransfer?.files;
  if (files && files.length > 0) {
    uploadFile(files[0]);
  }
};

const onFileSelect = (e: Event) => {
  const target = e.target as HTMLInputElement;
  if (target.files && target.files.length > 0) {
    uploadFile(target.files[0]);
  }
};

const uploadFile = (file: File) => {
  isUploading.value = true;
  uploadProgress.value = 0;
  uploadStatus.value = '';

  const formData = new FormData();
  formData.append('file', file);

  const xhr = new XMLHttpRequest();
  xhr.open('POST', '/api/upload', true);
  xhr.setRequestHeader('X-Client-ID', getClientID());
  xhr.setRequestHeader('X-Admin-Secret', getAdminSecret());

  xhr.upload.onprogress = (e) => {
    if (e.lengthComputable) {
      uploadProgress.value = Math.round((e.loaded / e.total) * 100);
    }
  };

  xhr.onload = () => {
    isUploading.value = false;
    uploadProgress.value = 0;
    if (xhr.status >= 200 && xhr.status < 300) {
      emit('uploaded');
    } else {
      try {
        const errorData = JSON.parse(xhr.responseText);
        uploadStatus.value = `Upload failed: ${errorData.error || xhr.statusText}`;
      } catch (e) {
        uploadStatus.value = `Upload failed: ${xhr.statusText}`;
      }
    }
  };

  xhr.onerror = () => {
    isUploading.value = false;
    uploadProgress.value = 0;
    uploadStatus.value = 'Error uploading file.';
  };

  xhr.send(formData);
};
</script>

<template>
  <div
    class="upload-zone p-5 border border-primary border-2 border-dashed rounded text-center mb-4 position-relative"
    :class="{ 'bg-light': isDragging || isUploading }"
    @dragover.prevent="!isUploading && (isDragging = true)"
    @dragleave.prevent="isDragging = false"
    @drop.prevent="onDrop"
  >
    <div v-if="!isUploading">
      <i class="ti ti-upload fs-1 text-primary mb-2"></i>
      <h4>Drag and drop files here</h4>
      <p>or</p>
      <label class="btn btn-primary cursor-pointer">
        Browse Files
        <input type="file" class="d-none" @change="onFileSelect" />
      </label>
      <p v-if="uploadStatus" class="mt-2 mb-0 text-danger small">{{ uploadStatus }}</p>
    </div>
    <div v-else class="py-3">
      <i class="ti ti-loader-2 fs-1 text-primary mb-2 spin"></i>
      <h4>Uploading...</h4>
      <div class="progress mt-3 mx-auto" style="max-width: 300px; height: 10px;">
        <div 
          class="progress-bar progress-bar-striped progress-bar-animated" 
          role="progressbar" 
          :style="{ width: uploadProgress + '%' }" 
          :aria-valuenow="uploadProgress" 
          aria-valuemin="0" 
          aria-valuemax="100"
        ></div>
      </div>
      <p class="mt-2 mb-0 small text-muted">{{ uploadProgress }}%</p>
    </div>
  </div>
</template>

<style scoped>
.upload-zone {
  cursor: pointer;
  transition: background-color 0.2s;
}
.border-dashed {
  border-style: dashed !important;
}
.cursor-pointer {
  cursor: pointer;
}
.spin {
  animation: spin 2s linear infinite;
}
@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
