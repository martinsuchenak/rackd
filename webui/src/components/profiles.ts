// Scan Profiles Management Components

export interface ScanProfile {
  id: string;
  name: string;
  description?: string;
  scan_type: string;
  enable_snmp: boolean;
  enable_ssh: boolean;
  ports: number[];
  timeout_sec: number;
  max_workers: number;
  created_at: string;
  updated_at: string;
}

interface ProfileFormData {
  id: string;
  name: string;
  description: string;
  scan_type: string;
  enable_snmp: boolean;
  enable_ssh: boolean;
  ports: number[];
  timeout_sec: number;
  max_workers: number;
}

export function profileList() {
  return {
    profiles: [] as ScanProfile[],
    loading: true,
    error: '',
    showModal: false,
    showDeleteModal: false,
    deleteTarget: null as ScanProfile | null,
    form: resetForm(),
    portsInput: '22,80,443,3389',

    async init() {
      await this.load();
    },

    async load() {
      this.loading = true;
      this.error = '';
      try {
        const response = await fetch('/api/scan-profiles');
        if (response.ok) {
          this.profiles = (await response.json()) || [];
        } else {
          this.error = 'Failed to load scan profiles';
        }
      } catch {
        this.error = 'Network error';
      } finally {
        this.loading = false;
      }
    },

    openAddModal() {
      this.form = resetForm();
      this.portsInput = '22,80,443,3389';
      this.showModal = true;
    },

    openEditModal(profile: ScanProfile) {
      this.form = {
        id: profile.id,
        name: profile.name,
        description: profile.description || '',
        scan_type: profile.scan_type,
        enable_snmp: profile.enable_snmp,
        enable_ssh: profile.enable_ssh,
        ports: profile.ports || [],
        timeout_sec: profile.timeout_sec,
        max_workers: profile.max_workers,
      };
      this.portsInput = (profile.ports || []).join(',');
      this.showModal = true;
    },

    closeModal() {
      this.showModal = false;
      this.form = resetForm();
      this.portsInput = '22,80,443,3389';
      this.error = '';
    },

    async save() {
      this.error = '';
      try {
        // Parse ports from string
        this.form.ports = this.portsInput
          .split(',')
          .map((p: string) => parseInt(p.trim(), 10))
          .filter((p: number) => !isNaN(p) && p > 0 && p <= 65535);

        const isEdit = !!this.form.id;
        const url = isEdit ? `/api/scan-profiles/${this.form.id}` : '/api/scan-profiles';
        const response = await fetch(url, {
          method: isEdit ? 'PUT' : 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(this.form),
        });

        if (response.ok) {
          this.closeModal();
          await this.load();
        } else {
          const data = await response.json();
          this.error = data.error || 'Failed to save profile';
        }
      } catch {
        this.error = 'Network error';
      }
    },

    confirmDelete(profile: ScanProfile) {
      this.deleteTarget = profile;
      this.showDeleteModal = true;
    },

    async deleteConfirmed() {
      if (!this.deleteTarget) return;
      try {
        const response = await fetch(`/api/scan-profiles/${this.deleteTarget.id}`, {
          method: 'DELETE',
        });
        if (response.ok) {
          this.showDeleteModal = false;
          this.deleteTarget = null;
          await this.load();
        } else {
          this.error = 'Failed to delete profile';
        }
      } catch {
        this.error = 'Network error';
      }
    },

    cancelDelete() {
      this.showDeleteModal = false;
      this.deleteTarget = null;
    },

    formatPorts(ports: number[]): string {
      if (!ports || ports.length === 0) return '-';
      if (ports.length <= 5) return ports.join(', ');
      return ports.slice(0, 5).join(', ') + ` (+${ports.length - 5})`;
    },
  };
}

function resetForm(): ProfileFormData {
  return {
    id: '',
    name: '',
    description: '',
    scan_type: 'quick',
    enable_snmp: false,
    enable_ssh: false,
    ports: [22, 80, 443, 3389],
    timeout_sec: 30,
    max_workers: 20,
  };
}

// Keep for backwards compatibility
export function profileForm() {
  return {};
}

// Page template for SPA
export function profilesPageTemplate(): string {
  return `
    <div x-data="profileList">
      <div class="flex justify-between items-center mb-6">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Scan Profiles</h1>
        <button x-show="$store.permissions.canCreate('scan_profiles')" @click="openAddModal()" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900 cursor-pointer transition-colors" aria-label="Add new scan profile">
          Add Profile
        </button>
      </div>

      <div x-show="error" role="alert" aria-live="polite" class="mb-4 p-4 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 rounded-md border border-red-300 dark:border-red-800" x-text="error"></div>

      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-300 dark:border-gray-700 overflow-hidden">
        <table class="min-w-full divide-y divide-gray-300 dark:divide-gray-700" role="table" aria-label="Scan profiles list">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th scope="col" class="px-6 py-3 text-left text-xs font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wider">Name</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wider">Type</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wider">Features</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wider">Ports</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wider">Settings</th>
              <th scope="col" class="px-6 py-3"><span class="sr-only">Actions</span></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-300 dark:divide-gray-700">
            <tr x-show="loading"><td colspan="6" class="px-6 py-8 text-center text-gray-600 dark:text-gray-400">Loading...</td></tr>
            <tr x-show="!loading && profiles.length === 0"><td colspan="6" class="px-6 py-8 text-center text-gray-600 dark:text-gray-400">No scan profiles found</td></tr>
            <template x-for="profile in profiles" :key="profile.id">
              <tr class="hover:bg-gray-50 dark:hover:bg-gray-700/50">
                <td class="px-6 py-4">
                  <button x-show="$store.permissions.canUpdate('scan_profiles')" @click="openEditModal(profile)" class="font-medium text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 rounded cursor-pointer transition-colors text-left" x-text="profile.name" :aria-label="'Edit profile: ' + profile.name"></button>
                  <span x-show="!$store.permissions.canUpdate('scan_profiles')" class="font-medium text-gray-900 dark:text-white" x-text="profile.name"></span>
                  <div class="text-sm text-gray-600 dark:text-gray-400" x-text="profile.description || ''"></div>
                </td>
                <td class="px-6 py-4">
                  <span class="px-2 py-1 text-xs font-medium rounded-full border"
                        :class="{
                          'bg-green-100 text-green-800 border-green-200 dark:bg-green-900/30 dark:text-green-400 dark:border-green-800': profile.scan_type === 'quick',
                          'bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:border-blue-800': profile.scan_type === 'full',
                          'bg-purple-100 text-purple-800 border-purple-200 dark:bg-purple-900/30 dark:text-purple-400 dark:border-purple-800': profile.scan_type === 'deep'
                        }"
                        x-text="profile.scan_type.toUpperCase()"></span>
                </td>
                <td class="px-6 py-4 text-sm text-gray-700 dark:text-gray-300">
                  <div class="flex gap-2">
                    <span x-show="profile.enable_snmp" class="px-2 py-0.5 text-xs bg-blue-100 text-blue-700 border border-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:border-blue-800 rounded">SNMP</span>
                    <span x-show="profile.enable_ssh" class="px-2 py-0.5 text-xs bg-green-100 text-green-700 border border-green-200 dark:bg-green-900/30 dark:text-green-400 dark:border-green-800 rounded">SSH</span>
                    <span x-show="!profile.enable_snmp && !profile.enable_ssh" class="text-gray-600 dark:text-gray-400">-</span>
                  </div>
                </td>
                <td class="px-6 py-4 text-sm text-gray-700 dark:text-gray-300" x-text="formatPorts(profile.ports)"></td>
                <td class="px-6 py-4 text-sm text-gray-700 dark:text-gray-300">
                  <div x-text="profile.timeout_sec + 's timeout'"></div>
                  <div x-text="profile.max_workers + ' workers'"></div>
                </td>
                <td class="px-6 py-4 text-right space-x-3">
                  <button x-show="$store.permissions.canUpdate('scan_profiles')" @click="openEditModal(profile)" class="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-800 dark:hover:text-blue-300 hover:underline focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 rounded cursor-pointer transition-colors" :aria-label="'Edit ' + profile.name">Edit</button>
                  <button x-show="$store.permissions.canDelete('scan_profiles')" @click="confirmDelete(profile)" class="text-sm text-red-600 dark:text-red-400 hover:text-red-800 dark:hover:text-red-300 hover:underline focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 rounded cursor-pointer transition-colors" :aria-label="'Delete ' + profile.name">Delete</button>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <!-- Profile Form Modal -->
      <div x-show="showModal" x-cloak class="fixed inset-0 z-50 overflow-y-auto" role="dialog" aria-modal="true" aria-labelledby="profile-modal-title">
        <div class="flex items-center justify-center min-h-screen px-4">
          <div class="fixed inset-0 bg-black/50" @click="closeModal()" aria-hidden="true"></div>
          <div class="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-lg w-full" role="document" @keydown.escape.window="closeModal()">
            <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200 dark:border-gray-700">
              <h2 id="profile-modal-title" class="text-lg font-semibold text-gray-900 dark:text-white" x-text="form.id ? 'Edit Profile' : 'Add Profile'"></h2>
              <button @click="closeModal()" class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500 rounded-full p-1 cursor-pointer transition-colors" aria-label="Close dialog">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
              </button>
            </div>
            <div class="p-6 space-y-4">
              <div>
                <label for="profile-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name <span class="text-red-500" aria-hidden="true">*</span></label>
                <input type="text" id="profile-name" x-model="form.name" required aria-required="true" class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors">
              </div>
              <div>
                <label for="profile-description" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
                <textarea id="profile-description" x-model="form.description" rows="2" class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors"></textarea>
              </div>
              <div>
                <label for="profile-scan-type" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Scan Type</label>
                <select id="profile-scan-type" x-model="form.scan_type" class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 cursor-pointer transition-colors">
                  <option value="quick">Quick - Basic ping scan</option>
                  <option value="full">Full - Extended port scan</option>
                  <option value="deep">Deep - Comprehensive with SNMP/SSH</option>
                </select>
              </div>
              <fieldset>
                <legend class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Protocol Features</legend>
                <div class="flex gap-6">
                  <label class="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" id="profile-enable-snmp" x-model="form.enable_snmp" class="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 cursor-pointer">
                    <span class="text-sm text-gray-700 dark:text-gray-300">Enable SNMP</span>
                  </label>
                  <label class="flex items-center gap-2 cursor-pointer">
                    <input type="checkbox" id="profile-enable-ssh" x-model="form.enable_ssh" class="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 cursor-pointer">
                    <span class="text-sm text-gray-700 dark:text-gray-300">Enable SSH</span>
                  </label>
                </div>
              </fieldset>
              <div>
                <label for="profile-ports" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Ports (comma separated)</label>
                <input type="text" id="profile-ports" x-model="portsInput" placeholder="22,80,443,3389" aria-describedby="ports-hint" class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors">
                <p id="ports-hint" class="text-xs text-gray-500 dark:text-gray-400 mt-1">Enter port numbers separated by commas, e.g., 22,80,443</p>
              </div>
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label for="profile-timeout" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Timeout (seconds)</label>
                  <input type="number" id="profile-timeout" x-model.number="form.timeout_sec" min="1" max="300" aria-describedby="timeout-hint" class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors">
                  <p id="timeout-hint" class="text-xs text-gray-500 dark:text-gray-400 mt-1">1-300 seconds</p>
                </div>
                <div>
                  <label for="profile-workers" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Max Workers</label>
                  <input type="number" id="profile-workers" x-model.number="form.max_workers" min="1" max="100" aria-describedby="workers-hint" class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 transition-colors">
                  <p id="workers-hint" class="text-xs text-gray-500 dark:text-gray-400 mt-1">1-100 concurrent</p>
                </div>
              </div>
            </div>
            <div class="flex justify-end gap-3 px-6 py-4 border-t border-gray-200 dark:border-gray-700">
              <button @click="closeModal()" type="button" class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 cursor-pointer transition-colors">Cancel</button>
              <button @click="save()" type="button" class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 cursor-pointer transition-colors">Save</button>
            </div>
          </div>
        </div>
      </div>

      <!-- Delete Confirmation Modal -->
      <div x-show="showDeleteModal" x-cloak class="fixed inset-0 z-50 overflow-y-auto" role="alertdialog" aria-modal="true" aria-labelledby="delete-profile-title" aria-describedby="delete-profile-desc">
        <div class="flex items-center justify-center min-h-screen px-4">
          <div class="fixed inset-0 bg-black/50" @click="cancelDelete()" aria-hidden="true"></div>
          <div class="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-sm w-full p-6" role="document" @keydown.escape.window="cancelDelete()">
            <button @click="cancelDelete()" class="absolute top-4 right-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500 rounded-full p-1 cursor-pointer transition-colors" aria-label="Close dialog">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
            </button>
            <h2 id="delete-profile-title" class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Delete Profile</h2>
            <p id="delete-profile-desc" class="text-gray-600 dark:text-gray-400 mb-6">Are you sure you want to delete "<span x-text="deleteTarget?.name"></span>"? This action cannot be undone.</p>
            <div class="flex justify-end gap-3">
              <button @click="cancelDelete()" type="button" class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-md focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 cursor-pointer transition-colors">Cancel</button>
              <button @click="deleteConfirmed()" type="button" class="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 dark:focus:ring-offset-gray-800 cursor-pointer transition-colors">Delete</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `;
}
