// Webhook Management Component

import type { Webhook, WebhookDelivery, EventTypeOption, EventType, CreateWebhookRequest, UpdateWebhookRequest } from '../core/types';
import { api, RackdAPIError } from '../core/api';

interface ValidationErrors {
  name?: string;
  url?: string;
  events?: string;
}

type ModalType = '' | 'create' | 'edit' | 'delete' | 'deliveries';

interface WebhookData {
  webhooks: Webhook[];
  deliveries: WebhookDelivery[];
  eventTypes: EventTypeOption[];
  loading: boolean;
  error: string;

  // Single modal state
  modalType: ModalType;
  selectedWebhook: Webhook | null;

  // Form data
  formData: {
    name: string;
    url: string;
    secret: string;
    events: EventType[];
    active: boolean;
    description: string;
  };
  validationErrors: ValidationErrors;
  saving: boolean;
  deleting: boolean;

  // Computed properties for template compatibility
  get showCreateModal(): boolean;
  get showEditModal(): boolean;
  get showDeleteModal(): boolean;
  get showDeliveriesModal(): boolean;
  get deleteModalTitle(): string;
  get deleteModalName(): string;
  get deleteModalDescription(): string;

  init(): Promise<void>;
  loadWebhooks(): Promise<void>;
  loadEventTypes(): Promise<void>;

  // Modal management
  openCreateModal(): void;
  openEditModal(webhook: Webhook): void;
  openDeleteModal(webhook: Webhook): void;
  closeModal(): void;
  closeDeleteModal(): void;
  closeDeliveriesModal(): void;
  cancelDelete(): void;

  // CRUD operations
  saveWebhook(): Promise<void>;
  doDelete(): Promise<void>;
  doDeleteWebhook(): Promise<void>;

  // Webhook actions
  pingWebhook(id: string): Promise<void>;
  viewDeliveries(webhook: Webhook): Promise<void>;

  // Form helpers
  isEventSelected(event: EventType): boolean;
  toggleEvent(event: EventType): void;
  validateForm(): boolean;

  // Utilities
  formatDate(dateStr: string): string;
  getEventLabel(event: string): string;
  getSelectedWebhookName(): string;
  hasWebhooks(): boolean;
}

export function webhookComponent(): WebhookData {
  return {
    webhooks: [],
    deliveries: [],
    eventTypes: [],
    loading: true,
    error: '',

    modalType: '',
    selectedWebhook: null,

    formData: {
      name: '',
      url: '',
      secret: '',
      events: [],
      active: true,
      description: ''
    },
    validationErrors: {},
    saving: false,
    deleting: false,

    // Computed getters for template compatibility
    get showCreateModal(): boolean { return this.modalType === 'create'; },
    get showEditModal(): boolean { return this.modalType === 'edit'; },
    get showDeleteModal(): boolean { return this.modalType === 'delete'; },
    get showDeliveriesModal(): boolean { return this.modalType === 'deliveries'; },
    get deleteModalTitle(): string { return 'Delete Webhook'; },
    get deleteModalName(): string { return this.getSelectedWebhookName(); },
    get deleteModalDescription(): string {
      return `Are you sure you want to delete ${this.getSelectedWebhookName()}? This action cannot be undone.`;
    },

    async init(): Promise<void> {
      await Promise.all([
        this.loadWebhooks(),
        this.loadEventTypes()
      ]);
    },

    async loadWebhooks(): Promise<void> {
      this.loading = true;
      try {
        this.webhooks = await api.listWebhooks();
      } catch (e) {
        console.error('Failed to load webhooks:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load webhooks';
      } finally {
        this.loading = false;
      }
    },

    async loadEventTypes(): Promise<void> {
      try {
        this.eventTypes = await api.getEventTypes();
      } catch (e) {
        console.error('Failed to load event types:', e);
      }
    },

    openCreateModal(): void {
      // Close any existing modal first
      this.modalType = '';
      this.selectedWebhook = null;
      this.deliveries = [];
      this.validationErrors = {};

      // Then open create modal
      this.formData = {
        name: '',
        url: '',
        secret: '',
        events: [],
        active: true,
        description: ''
      };
      this.modalType = 'create';
    },

    openEditModal(webhook: Webhook): void {
      // Close any existing modal first
      this.modalType = '';
      this.deliveries = [];
      this.validationErrors = {};

      // Then open edit modal
      this.selectedWebhook = webhook;
      this.formData = {
        name: webhook.name,
        url: webhook.url,
        secret: '',
        events: [...webhook.events],
        active: webhook.active,
        description: webhook.description || ''
      };
      this.modalType = 'edit';
    },

    openDeleteModal(webhook: Webhook): void {
      // Close any existing modal first
      this.modalType = '';
      this.deliveries = [];
      this.validationErrors = {};

      // Then open delete modal
      this.selectedWebhook = webhook;
      this.modalType = 'delete';
    },

    closeModal(): void {
      this.modalType = '';
      this.selectedWebhook = null;
      this.validationErrors = {};
    },

    closeDeleteModal(): void {
      this.modalType = '';
      this.selectedWebhook = null;
    },

    cancelDelete(): void {
      this.closeDeleteModal();
    },

    closeDeliveriesModal(): void {
      this.modalType = '';
      this.selectedWebhook = null;
      this.deliveries = [];
    },

    validateForm(): boolean {
      this.validationErrors = {};

      if (!this.formData.name.trim()) {
        this.validationErrors.name = 'Name is required';
      }

      if (!this.formData.url.trim()) {
        this.validationErrors.url = 'URL is required';
      } else {
        try {
          new URL(this.formData.url);
        } catch {
          this.validationErrors.url = 'Please enter a valid URL';
        }
      }

      if (this.formData.events.length === 0) {
        this.validationErrors.events = 'At least one event must be selected';
      }

      return Object.keys(this.validationErrors).length === 0;
    },

    async saveWebhook(): Promise<void> {
      if (!this.validateForm()) {
        return;
      }

      this.saving = true;
      this.error = '';

      try {
        if (this.selectedWebhook) {
          // Update existing webhook
          const updateData: UpdateWebhookRequest = {
            name: this.formData.name,
            url: this.formData.url,
            events: this.formData.events,
            active: this.formData.active,
            description: this.formData.description || undefined
          };
          if (this.formData.secret) {
            updateData.secret = this.formData.secret;
          }
          await api.updateWebhook(this.selectedWebhook.id, updateData);
        } else {
          // Create new webhook
          const createData: CreateWebhookRequest = {
            name: this.formData.name,
            url: this.formData.url,
            secret: this.formData.secret || undefined,
            events: this.formData.events,
            active: this.formData.active,
            description: this.formData.description || undefined
          };
          await api.createWebhook(createData);
        }
        await this.loadWebhooks();
        this.closeModal();
      } catch (e) {
        console.error('Failed to save webhook:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to save webhook';
      } finally {
        this.saving = false;
      }
    },

    async doDeleteWebhook(): Promise<void> {
      if (!this.selectedWebhook) return;

      this.deleting = true;
      try {
        await api.deleteWebhook(this.selectedWebhook.id);
        await this.loadWebhooks();
        this.closeDeleteModal();
      } catch (e) {
        console.error('Failed to delete webhook:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to delete webhook';
      } finally {
        this.deleting = false;
      }
    },

    async doDelete(): Promise<void> {
      await this.doDeleteWebhook();
    },

    async pingWebhook(id: string): Promise<void> {
      try {
        await api.pingWebhook(id);
        window.dispatchEvent(new CustomEvent('toast:success', {
          detail: { message: 'Test event sent successfully' }
        }));
      } catch (e) {
        console.error('Failed to ping webhook:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to send test event';
      }
    },

    async viewDeliveries(webhook: Webhook): Promise<void> {
      // Close any existing modal first
      this.modalType = '';
      this.deliveries = [];
      this.validationErrors = {};

      // Then open deliveries modal
      this.selectedWebhook = webhook;
      this.modalType = 'deliveries';
      try {
        this.deliveries = (await api.getWebhookDeliveries(webhook.id)) || [];
      } catch (e) {
        console.error('Failed to load deliveries:', e);
        this.deliveries = [];
      }
    },

    isEventSelected(event: EventType): boolean {
      return this.formData.events.includes(event);
    },

    toggleEvent(event: EventType): void {
      const index = this.formData.events.indexOf(event);
      if (index >= 0) {
        this.formData.events.splice(index, 1);
      } else {
        this.formData.events.push(event);
      }
    },

    formatDate(dateStr: string): string {
      if (!dateStr) return '';
      const date = new Date(dateStr);
      return date.toLocaleString();
    },

    getEventLabel(event: string): string {
      if (!event) return '';
      const parts = event.split('.');
      return parts.length > 1 ? parts[1] : event;
    },

    getSelectedWebhookName(): string {
      return this.selectedWebhook?.name || '';
    },

    hasWebhooks(): boolean {
      return this.webhooks.length > 0;
    }
  };
}
