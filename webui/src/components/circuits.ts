// Circuit Management Component

import type { Circuit, CircuitType, CircuitStatus, CircuitFilter, CreateCircuitRequest, UpdateCircuitRequest, Datacenter } from '../core/types';
import { api, RackdAPIError } from '../core/api';

interface ValidationErrors {
  name?: string;
  circuit_id?: string;
  provider?: string;
}

type ModalType = '' | 'create' | 'edit' | 'delete';

interface CircuitData {
  circuits: Circuit[];
  datacenters: Datacenter[];
  loading: boolean;
  error: string;

  // Filters
  filterProvider: string;
  filterStatus: string;
  filterType: string;

  // Single modal state
  modalType: ModalType;
  selectedCircuit: Circuit | null;

  // Form data
  formData: {
    name: string;
    circuit_id: string;
    provider: string;
    type: CircuitType;
    status: CircuitStatus;
    capacity_mbps: number;
    datacenter_a_id: string;
    datacenter_b_id: string;
    device_a_id: string;
    device_b_id: string;
    port_a: string;
    port_b: string;
    ip_address_a: string;
    ip_address_b: string;
    vlan_id: number;
    description: string;
    monthly_cost: number;
    contract_number: string;
    contact_name: string;
    contact_phone: string;
    contact_email: string;
  };
  validationErrors: ValidationErrors;
  saving: boolean;
  deleting: boolean;

  // Computed properties for template compatibility
  get showCreateModal(): boolean;
  get showEditModal(): boolean;
  get showDeleteModal(): boolean;

  init(): Promise<void>;
  loadCircuits(): Promise<void>;
  loadDatacenters(): Promise<void>;
  applyFilters(): Promise<void>;

  // Modal management
  openCreateModal(): void;
  openEditModal(circuit: Circuit): void;
  openDeleteModal(circuit: Circuit): void;
  closeModal(): void;
  closeDeleteModal(): void;

  // CRUD operations
  saveCircuit(): Promise<void>;
  doDeleteCircuit(): Promise<void>;

  // Form helpers
  validateForm(): boolean;
  getDatacenterName(id: string): string;

  getSelectedCircuitName(): string;
  getSelectedCircuitId(): string;

  // Utilities
  formatDate(dateStr: string): string;
  formatStatus(status: string): string;
  formatType(circuitType: string): string;

  // Helpers for CSP compliance
  getProviders(): string[];
  hasCircuits(): boolean;
}

export function circuitComponent(): CircuitData {
  return {
    circuits: [],
    datacenters: [],
    loading: true,
    error: '',

    filterProvider: '',
    filterStatus: '',
    filterType: '',

    modalType: '',
    selectedCircuit: null,

    formData: {
      name: '',
      circuit_id: '',
      provider: '',
      type: 'fiber',
      status: 'active',
      capacity_mbps: 0,
      datacenter_a_id: '',
      datacenter_b_id: '',
      device_a_id: '',
      device_b_id: '',
      port_a: '',
      port_b: '',
      ip_address_a: '',
      ip_address_b: '',
      vlan_id: 0,
      description: '',
      monthly_cost: 0,
      contract_number: '',
      contact_name: '',
      contact_phone: '',
      contact_email: ''
    },
    validationErrors: {},
    saving: false,
    deleting: false,

    // Computed getters for template compatibility
    get showCreateModal(): boolean { return this.modalType === 'create'; },
    get showEditModal(): boolean { return this.modalType === 'edit'; },
    get showDeleteModal(): boolean { return this.modalType === 'delete'; },

    async init(): Promise<void> {
      await Promise.all([
        this.loadCircuits(),
        this.loadDatacenters()
      ]);
    },

    async loadCircuits(): Promise<void> {
      this.loading = true;
      try {
        const filter: CircuitFilter = {};
        if (this.filterProvider) filter.provider = this.filterProvider;
        if (this.filterStatus) filter.status = this.filterStatus;
        if (this.filterType) filter.type = this.filterType;
        this.circuits = await api.listCircuits(Object.keys(filter).length > 0 ? filter : undefined);
      } catch (e) {
        console.error('Failed to load circuits:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load circuits';
      } finally {
        this.loading = false;
      }
    },

    async loadDatacenters(): Promise<void> {
      try {
        this.datacenters = await api.listDatacenters();
      } catch (e) {
        console.error('Failed to load datacenters:', e);
      }
    },

    async applyFilters(): Promise<void> {
      await this.loadCircuits();
    },

    openCreateModal(): void {
      this.modalType = '';
      this.selectedCircuit = null;
      this.validationErrors = {};

      this.formData = {
        name: '',
        circuit_id: '',
        provider: '',
        type: 'fiber',
        status: 'active',
        capacity_mbps: 0,
        datacenter_a_id: '',
        datacenter_b_id: '',
        device_a_id: '',
        device_b_id: '',
        port_a: '',
        port_b: '',
        ip_address_a: '',
        ip_address_b: '',
        vlan_id: 0,
        description: '',
        monthly_cost: 0,
        contract_number: '',
        contact_name: '',
        contact_phone: '',
        contact_email: ''
      };
      this.modalType = 'create';
    },

    openEditModal(circuit: Circuit): void {
      this.modalType = '';
      this.validationErrors = {};

      this.selectedCircuit = circuit;
      this.formData = {
        name: circuit.name,
        circuit_id: circuit.circuit_id,
        provider: circuit.provider,
        type: circuit.type,
        status: circuit.status,
        capacity_mbps: circuit.capacity_mbps || 0,
        datacenter_a_id: circuit.datacenter_a_id || '',
        datacenter_b_id: circuit.datacenter_b_id || '',
        device_a_id: circuit.device_a_id || '',
        device_b_id: circuit.device_b_id || '',
        port_a: circuit.port_a || '',
        port_b: circuit.port_b || '',
        ip_address_a: circuit.ip_address_a || '',
        ip_address_b: circuit.ip_address_b || '',
        vlan_id: circuit.vlan_id || 0,
        description: circuit.description || '',
        monthly_cost: circuit.monthly_cost || 0,
        contract_number: circuit.contract_number || '',
        contact_name: circuit.contact_name || '',
        contact_phone: circuit.contact_phone || '',
        contact_email: circuit.contact_email || ''
      };
      this.modalType = 'edit';
    },

    openDeleteModal(circuit: Circuit): void {
      this.modalType = '';
      this.validationErrors = {};
      this.selectedCircuit = circuit;
      this.modalType = 'delete';
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedCircuit = null;
      this.validationErrors = {};
    },

    closeDeleteModal(): void {
      this.modalType = '';
      this.selectedCircuit = null;
    },

    validateForm(): boolean {
      this.validationErrors = {};

      if (!this.formData.name.trim()) {
        this.validationErrors.name = 'Name is required';
      }

      if (!this.formData.circuit_id.trim()) {
        this.validationErrors.circuit_id = 'Circuit ID is required';
      }

      if (!this.formData.provider.trim()) {
        this.validationErrors.provider = 'Provider is required';
      }

      return Object.keys(this.validationErrors).length === 0;
    },

    async saveCircuit(): Promise<void> {
      if (!this.validateForm()) {
        return;
      }

      this.saving = true;
      this.error = '';

      try {
        if (this.selectedCircuit) {
          // Update existing circuit
          const updateData: UpdateCircuitRequest = {
            name: this.formData.name,
            circuit_id: this.formData.circuit_id,
            provider: this.formData.provider,
            type: this.formData.type,
            status: this.formData.status,
            capacity_mbps: this.formData.capacity_mbps || undefined,
            datacenter_a_id: this.formData.datacenter_a_id || undefined,
            datacenter_b_id: this.formData.datacenter_b_id || undefined,
            device_a_id: this.formData.device_a_id || undefined,
            device_b_id: this.formData.device_b_id || undefined,
            port_a: this.formData.port_a || undefined,
            port_b: this.formData.port_b || undefined,
            ip_address_a: this.formData.ip_address_a || undefined,
            ip_address_b: this.formData.ip_address_b || undefined,
            vlan_id: this.formData.vlan_id || undefined,
            description: this.formData.description || undefined,
            monthly_cost: this.formData.monthly_cost || undefined,
            contract_number: this.formData.contract_number || undefined,
            contact_name: this.formData.contact_name || undefined,
            contact_phone: this.formData.contact_phone || undefined,
            contact_email: this.formData.contact_email || undefined
          };
          await api.updateCircuit(this.selectedCircuit.id, updateData);
        } else {
          // Create new circuit
          const createData: CreateCircuitRequest = {
            name: this.formData.name,
            circuit_id: this.formData.circuit_id,
            provider: this.formData.provider,
            type: this.formData.type,
            status: this.formData.status,
            capacity_mbps: this.formData.capacity_mbps || undefined,
            datacenter_a_id: this.formData.datacenter_a_id || undefined,
            datacenter_b_id: this.formData.datacenter_b_id || undefined,
            device_a_id: this.formData.device_a_id || undefined,
            device_b_id: this.formData.device_b_id || undefined,
            port_a: this.formData.port_a || undefined,
            port_b: this.formData.port_b || undefined,
            ip_address_a: this.formData.ip_address_a || undefined,
            ip_address_b: this.formData.ip_address_b || undefined,
            vlan_id: this.formData.vlan_id || undefined,
            description: this.formData.description || undefined,
            monthly_cost: this.formData.monthly_cost || undefined,
            contract_number: this.formData.contract_number || undefined,
            contact_name: this.formData.contact_name || undefined,
            contact_phone: this.formData.contact_phone || undefined,
            contact_email: this.formData.contact_email || undefined
          };
          await api.createCircuit(createData);
        }
        await this.loadCircuits();
        this.closeModal();
      } catch (e) {
        console.error('Failed to save circuit:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save circuit';
      } finally {
        this.saving = false;
      }
    },

    async doDeleteCircuit(): Promise<void> {
      if (!this.selectedCircuit) return;

      this.deleting = true;
      try {
        await api.deleteCircuit(this.selectedCircuit.id);
        await this.loadCircuits();
        this.closeDeleteModal();
      } catch (e) {
        console.error('Failed to delete circuit:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete circuit';
      } finally {
        this.deleting = false;
      }
    },

    getDatacenterName(id: string): string {
      if (!id) return '-';
      const dc = this.datacenters.find(d => d.id === id);
      return dc ? dc.name : id;
    },

    formatDate(dateStr: string): string {
      if (!dateStr) return '';
      const date = new Date(dateStr);
      return date.toLocaleString();
    },

    formatStatus(status: string): string {
      const statusMap: Record<string, string> = {
        'active': 'Active',
        'inactive': 'Inactive',
        'planned': 'Planned',
        'decommissioned': 'Decommissioned'
      };
      return statusMap[status] || status;
    },

    formatType(circuitType: string): string {
      const typeMap: Record<string, string> = {
        'fiber': 'Fiber',
        'copper': 'Copper',
        'microwave': 'Microwave',
        'dark_fiber': 'Dark Fiber'
      };
      return typeMap[circuitType] || circuitType;
    },

    getProviders(): string[] {
      if (!this.circuits) return [];
      const providers = this.circuits.map(c => c.provider).filter(p => !!p);
      return Array.from(new Set(providers));
    },

    hasCircuits(): boolean {
      return !!(this.circuits && this.circuits.length > 0);
    },

    getSelectedCircuitName(): string {
      return this.selectedCircuit ? this.selectedCircuit.name : '';
    },

    getSelectedCircuitId(): string {
      return this.selectedCircuit ? this.selectedCircuit.id : '';
    }
  };
}
