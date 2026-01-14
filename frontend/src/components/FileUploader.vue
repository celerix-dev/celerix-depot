<script setup lang="ts">
import { ref } from 'vue';
import { getClientID, getAdminSecret } from '@/utils/persona';

const emit = defineEmits(['uploaded']);
const isDragging = ref(false);
const uploadStatus = ref('');

const onDrop = (e: DragEvent) => {
  isDragging.value = false;
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

const uploadFile = async (file: File) => {
  uploadStatus.value = `Uploading ${file.name}...`;
  const formData = new FormData();
  formData.append('file', file);

  try {
    const response = await fetch('/api/upload', {
      method: 'POST',
      body: formData,
      headers: {
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
    });

    if (response.ok) {
      uploadStatus.value = 'Upload successful!';
      emit('uploaded');
    } else {
      const errorData = await response.json().catch(() => ({}));
      uploadStatus.value = `Upload failed: ${errorData.error || response.statusText}`;
    }
  } catch (error) {
    console.error('Error uploading file:', error);
    uploadStatus.value = 'Error uploading file.';
  }
};
</script>

<template>
  <div
    class="upload-zone p-5 border border-primary border-2 border-dashed rounded text-center mb-4"
    :class="{ 'bg-light': isDragging }"
    @dragover.prevent="isDragging = true"
    @dragleave.prevent="isDragging = false"
    @drop.prevent="onDrop"
  >
    <div v-if="!uploadStatus">
      <i class="ti ti-upload fs-1 text-primary mb-2"></i>
      <h4>Drag and drop files here</h4>
      <p>or</p>
      <label class="btn btn-primary cursor-pointer">
        Browse Files
        <input type="file" class="d-none" @change="onFileSelect" />
      </label>
    </div>
    <div v-else>
      <p class="mb-0">{{ uploadStatus }}</p>
      <button v-if="uploadStatus.includes('successful') || uploadStatus.includes('failed')" 
              class="btn btn-sm btn-link mt-2" @click="uploadStatus = ''">Upload another</button>
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
</style>
