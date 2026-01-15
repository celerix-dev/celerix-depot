<script setup lang="ts">
import FileUploader from '@/components/FileUploader.vue';
import FileList from '@/components/FileList.vue';
import { onMounted, ref } from 'vue';
import { fetchPersona, updateClientName, activateAdmin, recoverPersona } from '@/utils/persona';

import logo from '@/assets/celerix-logo.png';

const fileListRef = ref<InstanceType<typeof FileList> | null>(null);
const persona = ref('client');
const clientName = ref('');
const recoveryCode = ref('');
const appVersion = ref('');
const showNamingModal = ref(false);
const showAdminModal = ref(false);
const showRecoveryModal = ref(false);
const showRecoveryNoticeModal = ref(false);
const hasNotedRecoveryCode = ref(false);
const recoveryInput = ref('');
const newName = ref('');
const adminSecret = ref('');

const onUploaded = () => {
  console.log('File uploaded successfully, refreshing file list...');
  if (fileListRef.value) {
    fileListRef.value.fetchFiles();
  } else {
    console.error('fileListRef is null!');
  }
};

const saveName = async () => {
  if (newName.value.trim()) {
    const result = await updateClientName(newName.value.trim());
    if (result.success) {
      clientName.value = newName.value.trim();
      recoveryCode.value = result.recovery_code || '';
      showNamingModal.value = false;
      showRecoveryNoticeModal.value = true;
    }
  }
};

const loginAdmin = async () => {
  if (adminSecret.value.trim()) {
    const result = await activateAdmin(adminSecret.value.trim());
    if (result.success) {
      const data = await fetchPersona();
      persona.value = data.persona;
      clientName.value = data.name;
      recoveryCode.value = data.recovery_code || '';
      showAdminModal.value = false;
      adminSecret.value = '';
      if (fileListRef.value) {
        fileListRef.value.fetchFiles();
      }
    } else {
      alert('Invalid admin secret.');
    }
  }
};

const performRecovery = async () => {
  if (recoveryInput.value.trim()) {
    const result = await recoverPersona(recoveryInput.value.trim());
    if (result.success) {
      persona.value = result.persona || 'client';
      clientName.value = result.name || '';
      showRecoveryModal.value = false;
      recoveryInput.value = '';
      const data = await fetchPersona();
      recoveryCode.value = data.recovery_code || '';
      if (fileListRef.value) {
        fileListRef.value.fetchFiles();
      }
    } else {
      alert('Invalid recovery code.');
    }
  }
};

const closeRecoveryModal = () => {
  showRecoveryModal.value = false;
  if (persona.value === 'client' && !clientName.value) {
    showNamingModal.value = true;
  }
};

const refreshPersona = async () => {
  const data = await fetchPersona();
  persona.value = data.persona;
  clientName.value = data.name;
  recoveryCode.value = data.recovery_code || '';
  appVersion.value = data.version || '';

  if (persona.value === 'client' && !clientName.value) {
    showNamingModal.value = true;
  }
};

onMounted(refreshPersona);
</script>

<template>
  <div class="container py-5">
    <div class="row justify-content-center">
      <div class="col-lg-10">
        <div class="d-flex justify-content-between align-items-center mb-4">
          <div class="d-flex align-items-center">
            <div class="d-flex align-items-center">
              <div><img :src="logo" style="width:36px;height:36px;" alt="Celerix logo"></div>
              <div><h2 class="ms-2 mb-0 me-3">Celerix Depot</h2></div>
              <div v-if="appVersion"><small class="text-muted">v{{ appVersion }}</small></div>
            </div>
          </div>
          <div class="d-flex align-items-center">
            <span :class="['badge', persona === 'admin' ? 'bg-danger' : 'bg-info', 'me-2']">
              {{ persona.toUpperCase() }} PERSONA <span v-if="clientName">: {{ clientName}}</span>
            </span>
            <router-link v-if="persona === 'admin'" to="/admin" class="btn btn-sm btn-outline-danger me-2" title="Admin Management">
              <i class="ti ti-database"></i>
            </router-link>
            <button v-if="persona !== 'admin'" class="btn btn-sm btn-outline-secondary me-2" @click="showAdminModal = true" title="Admin Access">
              <i class="ti ti-settings"></i>
            </button>
            <button class="btn btn-sm btn-outline-secondary" @click="showRecoveryModal = true" title="Recover Persona">
              <i class="ti ti-refresh"></i>
            </button>
          </div>
        </div>
        <FileUploader @uploaded="onUploaded" />
        <FileList ref="fileListRef" :persona="persona" />
      </div>
    </div>

    <!-- Naming Modal -->
    <div v-if="showNamingModal" class="modal d-block" style="background: rgba(0,0,0,0.5)">
      <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Identify Yourself</h5>
          </div>
          <div class="modal-body">
            <p>Please enter your name to continue.</p>
            <input v-model="newName" type="text" class="form-control" placeholder="Your Name" @keyup.enter="saveName" />
          </div>
          <div class="modal-footer d-flex flex-column">
            <button class="btn btn-primary w-100 mb-2" :disabled="!newName.trim()" @click="saveName">Save Name</button>
            <button class="btn btn-link btn-sm" @click="showNamingModal = false; showRecoveryModal = true">Already have a persona? Restore here</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Admin Login Modal -->
    <div v-if="showAdminModal" class="modal d-block" style="background: rgba(0,0,0,0.5)">
      <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Admin Access</h5>
            <button type="button" class="btn-close" @click="showAdminModal = false"></button>
          </div>
          <div class="modal-body">
            <p>Enter the Admin Secret to switch to Admin Persona.</p>
            <input v-model="adminSecret" type="password" class="form-control" placeholder="Secret Key" @keyup.enter="loginAdmin" />
          </div>
          <div class="modal-footer">
            <button class="btn btn-danger w-100" :disabled="!adminSecret.trim()" @click="loginAdmin">Login</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Recovery Notice Modal -->
    <div v-if="showRecoveryNoticeModal" class="modal d-block" style="background: rgba(0,0,0,0.5)">
      <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content border-warning border-2">
          <div class="modal-header bg-warning text-dark">
            <h5 class="modal-title">Important: Save Your Recovery Code</h5>
          </div>
          <div class="modal-body">
            <p>Your persona has been created. Please save this recovery code. You will need it to restore your access if you clear your browser data or use a different computer.</p>
            <div class="alert alert-secondary text-center">
              <code class="fs-4">{{ recoveryCode }}</code>
            </div>
            <div class="form-check mt-3">
              <input v-model="hasNotedRecoveryCode" class="form-check-input" type="checkbox" id="checkRecovery" />
              <label class="form-check-label" for="checkRecovery">
                I have made a note of the recovery code
              </label>
            </div>
          </div>
          <div class="modal-footer">
            <button class="btn btn-primary w-100" :disabled="!hasNotedRecoveryCode" @click="showRecoveryNoticeModal = false">Continue</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Recovery Modal -->
    <div v-if="showRecoveryModal" class="modal d-block" style="background: rgba(0,0,0,0.5)">
      <div class="modal-dialog modal-dialog-centered">
        <div class="modal-content">
          <div class="modal-header">
            <h5 class="modal-title">Recover Persona</h5>
            <button type="button" class="btn-close" @click="closeRecoveryModal"></button>
          </div>
          <div class="modal-body">
            <p>Enter your recovery code to restore your persona.</p>
            <input v-model="recoveryInput" type="text" class="form-control" placeholder="Recovery Code" @keyup.enter="performRecovery" />
          </div>
          <div class="modal-footer">
            <button class="btn btn-primary w-100" :disabled="!recoveryInput.trim()" @click="performRecovery">Recover</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
</style>