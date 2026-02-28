import { api } from '../core/api';
import type { NATMapping, NATFilter, CreateNATRequest, UpdateNATRequest, Device, Datacenter, Network } from '../core/types';

export function natComponent() {
  return {
    mappings: [] as NATMapping[],
    loading: true,
    error: '',
    showForm: false,
    showDeleteConfirm: false,
    editingId: null as string | null,
    deleteId: null as string | null,
    form: {
      name: '',
      external_ip: '',
      external_port: '' as string,
      internal_ip: '',
      internal_port: '' as string,
      protocol: 'tcp' as 'tcp' | 'udp' | 'any',
      device_id: '',
      description: '',
      enabled: true,
      datacenter_id: '',
      network_id: '',
      tags: '',
    },
    formErrors: {} as Record<string, string>,
    saving: false,
    filters: {} as NATFilter,
    showFilters: false,
    devices: [] as Device[],
    datacenters: [] as Datacenter[],
    networks: [] as Network[],
    loadingDevices: false,
    loadingDatacenters: false,
    loadingNetworks: false,

    async init() {
      await this.loadMappings();
      await this.loadDevices();
      await this.loadDatacenters();
      await this.loadNetworks();
    },

    async loadMappings() {
      this.loading = true;
      this.error = '';
      try {
        this.mappings = await api.listNATMappings(this.filters);
      } catch (e: any) {
        this.error = e.message || 'Failed to load NAT mappings';
      } finally {
        this.loading = false;
      }
    },

    async loadDevices() {
      this.loadingDevices = true;
      try {
        this.devices = await api.listDevices();
      } catch {
        // Non-critical error
      } finally {
        this.loadingDevices = false;
      }
    },

    async loadDatacenters() {
      this.loadingDatacenters = true;
      try {
        this.datacenters = await api.listDatacenters();
      } catch {
        // Non-critical error
      } finally {
        this.loadingDatacenters = false;
      }
    },

    async loadNetworks() {
      this.loadingNetworks = true;
      try {
        this.networks = await api.listNetworks();
      } catch {
        // Non-critical error
      } finally {
        this.loadingNetworks = false;
      }
    },

    openCreateForm() {
      this.editingId = null;
      this.form = {
        name: '',
        external_ip: '',
        external_port: '',
        internal_ip: '',
        internal_port: '',
        protocol: 'tcp',
        device_id: '',
        description: '',
        enabled: true,
        datacenter_id: '',
        network_id: '',
        tags: '',
      };
      this.formErrors = {};
      this.showForm = true;
    },

    openEditForm(mapping: NATMapping) {
      this.editingId = mapping.id;
      this.form = {
        name: mapping.name,
        external_ip: mapping.external_ip,
        external_port: String(mapping.external_port),
        internal_ip: mapping.internal_ip,
        internal_port: String(mapping.internal_port),
        protocol: mapping.protocol,
        device_id: mapping.device_id || '',
        description: mapping.description || '',
        enabled: mapping.enabled,
        datacenter_id: mapping.datacenter_id || '',
        network_id: mapping.network_id || '',
        tags: (mapping.tags || []).join(', '),
      };
      this.formErrors = {};
      this.showForm = true;
    },

    closeForm() {
      this.showForm = false;
      this.editingId = null;
      this.formErrors = {};
    },

    validateForm(): boolean {
      this.formErrors = {};

      if (!this.form.name.trim()) {
        this.formErrors.name = 'Name is required';
      }
      if (!this.form.external_ip.trim()) {
        this.formErrors.external_ip = 'External IP is required';
      }
      if (!this.form.internal_ip.trim()) {
        this.formErrors.internal_ip = 'Internal IP is required';
      }
      const extPort = Number(this.form.external_port);
      const intPort = Number(this.form.internal_port);
      if (this.form.external_port === '' || isNaN(extPort) || extPort < 0 || extPort > 65535) {
        this.formErrors.external_port = 'Port must be between 0 and 65535';
      }
      if (this.form.internal_port === '' || isNaN(intPort) || intPort < 0 || intPort > 65535) {
        this.formErrors.internal_port = 'Port must be between 0 and 65535';
      }

      return Object.keys(this.formErrors).length === 0;
    },

    async saveForm() {
      if (!this.validateForm()) return;

      this.saving = true;
      try {
        const tags = this.form.tags
          .split(',')
          .map(t => t.trim())
          .filter(t => t);

        const data: CreateNATRequest | UpdateNATRequest = {
          name: this.form.name.trim(),
          external_ip: this.form.external_ip.trim(),
          external_port: Number(this.form.external_port),
          internal_ip: this.form.internal_ip.trim(),
          internal_port: Number(this.form.internal_port),
          protocol: this.form.protocol,
          device_id: this.form.device_id || undefined,
          description: this.form.description.trim(),
          enabled: this.form.enabled,
          datacenter_id: this.form.datacenter_id || undefined,
          network_id: this.form.network_id || undefined,
          tags,
        };

        if (this.editingId) {
          await api.updateNATMapping(this.editingId, data);
        } else {
          await api.createNATMapping(data as CreateNATRequest);
        }

        this.closeForm();
        await this.loadMappings();
      } catch (e: any) {
        if (e.data?.errors) {
          this.formErrors = e.data.errors;
        } else {
          this.error = e.message || 'Failed to save NAT mapping';
        }
      } finally {
        this.saving = false;
      }
    },

    confirmDelete(id: string) {
      this.deleteId = id;
      this.showDeleteConfirm = true;
    },

    cancelDelete() {
      this.showDeleteConfirm = false;
      this.deleteId = null;
    },

    async deleteMapping() {
      if (!this.deleteId) return;

      try {
        await api.deleteNATMapping(this.deleteId);
        this.showDeleteConfirm = false;
        this.deleteId = null;
        await this.loadMappings();
      } catch (e: any) {
        this.error = e.message || 'Failed to delete NAT mapping';
      }
    },

    async applyFilters() {
      await this.loadMappings();
      this.showFilters = false;
    },

    clearFilters() {
      this.filters = {};
      this.loadMappings();
      this.showFilters = false;
    },

    getDeviceName(deviceId: string): string {
      if (!deviceId) return '';
      const device = this.devices.find(d => d.id === deviceId);
      return device ? device.name : deviceId;
    },

    getDatacenterName(datacenterId: string): string {
      if (!datacenterId) return '';
      const dc = this.datacenters.find(d => d.id === datacenterId);
      return dc ? dc.name : datacenterId;
    },

    getNetworkName(networkId: string): string {
      if (!networkId) return '';
      const network = this.networks.find(n => n.id === networkId);
      return network ? network.name : networkId;
    },

    formatProtocol(protocol: string): string {
      return protocol.toUpperCase();
    },

    formatAddress(ip: string, port: number): string {
      return `${ip}:${port}`;
    },
  };
}
