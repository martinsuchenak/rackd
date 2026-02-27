// Custom Field Management Component

import type { CustomFieldDefinition, CustomFieldType, CreateCustomFieldDefinitionRequest, UpdateCustomFieldDefinitionRequest } from '../core/types';
import { api, RackdAPIError } from '../core/api';

interface ValidationErrors {
  name?: string;
  key?: string;
  type?: string;
  options?: string;
}

type ModalType = '' | 'create' | 'edit' | 'delete';

interface CustomFieldData {
  definitions: CustomFieldDefinition[];
  fieldTypes: { value: string; label: string }[];
  loading: boolean;
  error: string;

  // Single modal state
  modalType: ModalType;
  selectedField: CustomFieldDefinition | null;

  // Form data
  formData: {
    name: string;
    key: string;
    type: CustomFieldType;
    required: boolean;
    options: string;
    description: string;
  };
  validationErrors: ValidationErrors;
  saving: boolean;
  deleting: boolean;

  // Computed properties for template compatibility
  get showCreateModal(): boolean;
  get showEditModal(): boolean;
  get showDeleteModal(): boolean;

  init(): Promise<void>;
  loadDefinitions(): Promise<void>;
  loadFieldTypes(): Promise<void>;

  // Modal management
  openCreateModal(): void;
  openEditModal(field: CustomFieldDefinition): void;
  openDeleteModal(field: CustomFieldDefinition): void;
  closeModal(): void;
  closeDeleteModal(): void;

  // CRUD operations
  saveField(): Promise<void>;
  doDeleteField(): Promise<void>;

  // Form helpers
  validateForm(): boolean;
  isSelectType(): boolean;

  // Utilities
  formatDate(dateStr: string): string;
  getTypeLabel(type: CustomFieldType): string;
}

export function customFieldComponent(): CustomFieldData {
  return {
    definitions: [],
    fieldTypes: [],
    loading: true,
    error: '',

    modalType: '',
    selectedField: null,

    formData: {
      name: '',
      key: '',
      type: 'text',
      required: false,
      options: '',
      description: ''
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
        this.loadDefinitions(),
        this.loadFieldTypes()
      ]);
    },

    async loadDefinitions(): Promise<void> {
      this.loading = true;
      try {
        this.definitions = await api.listCustomFieldDefinitions();
      } catch (e) {
        console.error('Failed to load custom field definitions:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load custom field definitions';
      } finally {
        this.loading = false;
      }
    },

    async loadFieldTypes(): Promise<void> {
      try {
        this.fieldTypes = await api.getCustomFieldTypes();
      } catch (e) {
        console.error('Failed to load field types:', e);
      }
    },

    openCreateModal(): void {
      // Close any existing modal first
      this.modalType = '';
      this.selectedField = null;
      this.validationErrors = {};

      // Then open create modal
      this.formData = {
        name: '',
        key: '',
        type: 'text',
        required: false,
        options: '',
        description: ''
      };
      this.modalType = 'create';
    },

    openEditModal(field: CustomFieldDefinition): void {
      // Close any existing modal first
      this.modalType = '';
      this.validationErrors = {};

      // Then open edit modal
      this.selectedField = field;
      this.formData = {
        name: field.name,
        key: field.key,
        type: field.type,
        required: field.required,
        options: field.options ? field.options.join(', ') : '',
        description: field.description || ''
      };
      this.modalType = 'edit';
    },

    openDeleteModal(field: CustomFieldDefinition): void {
      this.modalType = '';
      this.selectedField = field;
      this.modalType = 'delete';
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedField = null;
      this.validationErrors = {};
    },

    closeDeleteModal(): void {
      this.modalType = '';
      this.selectedField = null;
    },

    isSelectType(): boolean {
      return this.formData.type === 'select';
    },

    validateForm(): boolean {
      this.validationErrors = {};

      if (!this.formData.name.trim()) {
        this.validationErrors.name = 'Name is required';
      }

      if (!this.formData.key.trim()) {
        this.validationErrors.key = 'Key is required';
      } else if (!/^[a-z0-9_]+$/.test(this.formData.key)) {
        this.validationErrors.key = 'Key must contain only lowercase letters, numbers, and underscores';
      }

      if (this.formData.type === 'select' && !this.formData.options.trim()) {
        this.validationErrors.options = 'Options are required for select type';
      }

      return Object.keys(this.validationErrors).length === 0;
    },

    async saveField(): Promise<void> {
      if (!this.validateForm()) {
        return;
      }

      this.saving = true;
      try {
        const options = this.formData.type === 'select'
          ? this.formData.options.split(',').map(o => o.trim()).filter(o => o)
          : [];

        if (this.modalType === 'create') {
          const request: CreateCustomFieldDefinitionRequest = {
            name: this.formData.name.trim(),
            key: this.formData.key.trim(),
            type: this.formData.type,
            required: this.formData.required,
            options: options.length > 0 ? options : undefined,
            description: this.formData.description.trim() || undefined
          };
          await api.createCustomFieldDefinition(request);
        } else if (this.modalType === 'edit' && this.selectedField) {
          const request: UpdateCustomFieldDefinitionRequest = {
            name: this.formData.name.trim(),
            key: this.formData.key.trim(),
            type: this.formData.type,
            required: this.formData.required,
            options: options.length > 0 ? options : undefined,
            description: this.formData.description.trim() || undefined
          };
          await api.updateCustomFieldDefinition(this.selectedField.id, request);
        }

        await this.loadDefinitions();
        this.closeModal();
      } catch (e) {
        console.error('Failed to save custom field:', e);
        if (e instanceof RackdAPIError) {
          if (e.details && 'field' in e.details && 'message' in e.details) {
            const field = e.details.field as string;
            if (field === 'key') {
              this.validationErrors.key = e.message;
            } else if (field === 'name') {
              this.validationErrors.name = e.message;
            } else {
              this.validationErrors.key = e.message;
            }
          } else {
            this.validationErrors.key = e.message;
          }
        }
      } finally {
        this.saving = false;
      }
    },

    async doDeleteField(): Promise<void> {
      if (!this.selectedField) return;

      this.deleting = true;
      try {
        await api.deleteCustomFieldDefinition(this.selectedField.id);
        await this.loadDefinitions();
        this.closeDeleteModal();
      } catch (e) {
        console.error('Failed to delete custom field:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete custom field';
      } finally {
        this.deleting = false;
      }
    },

    formatDate(dateStr: string): string {
      if (!dateStr) return '';
      const date = new Date(dateStr);
      return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
    },

    getTypeLabel(type: CustomFieldType): string {
      const labels: Record<CustomFieldType, string> = {
        'text': 'Text',
        'number': 'Number',
        'boolean': 'Boolean',
        'select': 'Select'
      };
      return labels[type] || type;
    }
  };
}
