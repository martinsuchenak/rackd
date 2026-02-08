// Roles Component for Rackd Web UI

import type { Role, Permission, RoleFilter, CreateRoleRequest, UpdateRoleRequest } from '../core/types';
import { api, RackdAPIError } from '../core/api';

interface RolesListData {
  roles: Role[];
  permissions: Permission[];
  loading: boolean;
  error: string;
  filter: RoleFilter;
  page: number;
  pageSize: number;
  totalPages: number;
  pagedRoles: Role[];
  showCreateModal: boolean;
  showEditModal: boolean;
  showDeleteModal: boolean;
  selectedRole: Role | null;
  showPermissionsModal: boolean;
  saving: boolean;
  deleting: boolean;
  validationErrors: Record<string, string>;
  createForm: CreateRoleRequest;
  editForm: UpdateRoleRequest;
  availablePermissions: Permission[];
  selectedPermissionIds: string[];
  init(): Promise<void>;
  loadRoles(): Promise<void>;
  loadPermissions(): Promise<void>;
  applyFilter(): void;
  clearFilter(): void;
  goToPage(p: number): void;
  openCreateModal(): void;
  closeCreateModal(): void;
  openEditModal(role: Role): void;
  closeEditModal(): void;
  openDeleteModal(role: Role): void;
  closeDeleteModal(): void;
  openPermissionsModal(role: Role): void;
  closePermissionsModal(): void;
  doCreateRole(): Promise<void>;
  doUpdateRole(): Promise<void>;
  doDeleteRole(): Promise<void>;
  togglePermission(permissionId: string): void;
}

export function rolesList() {
  return {
    roles: [] as Role[],
    permissions: [] as Permission[],
    loading: true,
    error: '',
    filter: {} as RoleFilter,
    page: 1,
    pageSize: 10,
    showCreateModal: false,
    showEditModal: false,
    showDeleteModal: false,
    showPermissionsModal: false,
    selectedRole: null as Role | null,
    saving: false,
    deleting: false,
    validationErrors: {} as Record<string, string>,
    createForm: {
      name: '',
      description: '',
      permissions: [],
    } as CreateRoleRequest,
    editForm: {
      description: '',
      permissions: [],
    } as UpdateRoleRequest,
    availablePermissions: [] as Permission[],
    selectedPermissionIds: [] as string[],

    get totalPages(): number {
      return Math.ceil(this.roles.length / this.pageSize) || 1;
    },

    get pagedRoles(): Role[] {
      const start = (this.page - 1) * this.pageSize;
      return this.roles.slice(start, start + this.pageSize);
    },

    get deleteModalTitle(): string {
      return this.selectedRole?.is_system ? 'Cannot Delete System Role' : 'Delete Role';
    },

    get deleteModalMessage(): string {
      if (this.selectedRole?.is_system) {
        return 'System roles cannot be deleted or modified.';
      }
      return `Are you sure you want to delete the role "${this.selectedRole?.name}"? This action cannot be undone.`;
    },

    get deleteDisabled(): boolean {
      return this.selectedRole?.is_system || false;
    },

    get editDisabled(): boolean {
      return this.selectedRole?.is_system || false;
    },

    async init(): Promise<void> {
      await Promise.all([this.loadRoles(), this.loadPermissions()]);
    },

    async loadRoles(): Promise<void> {
      this.loading = true;
      this.error = '';

      try {
        this.roles = await api.listRoles(this.filter);
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to load roles';
        }
      } finally {
        this.loading = false;
      }
    },

    async loadPermissions(): Promise<void> {
      try {
        this.availablePermissions = await api.listPermissions();
      } catch {
        this.availablePermissions = [];
      }
    },

    applyFilter(): void {
      this.page = 1;
      this.loadRoles();
    },

    clearFilter(): void {
      this.filter = {} as RoleFilter;
      this.page = 1;
      this.loadRoles();
    },

    goToPage(p: number): void {
      this.page = p;
    },

    openCreateModal(): void {
      this.showCreateModal = true;
      this.validationErrors = {};
      this.createForm = {
        name: '',
        description: '',
        permissions: [],
      };
      this.selectedPermissionIds = [];
    },

    closeCreateModal(): void {
      this.showCreateModal = false;
      this.validationErrors = {};
      this.createForm = {
        name: '',
        description: '',
        permissions: [],
      };
      this.selectedPermissionIds = [];
    },

    openEditModal(role: Role): void {
      if (role.is_system) {
        this.error = 'Cannot edit system roles';
        return;
      }

      this.showEditModal = true;
      this.selectedRole = role;
      this.validationErrors = {};
      this.editForm = {
        description: role.description || '',
        permissions: role.permissions ? role.permissions.map(p => p.id) : [],
      };
      this.selectedPermissionIds = role.permissions ? role.permissions.map(p => p.id) : [];
    },

    closeEditModal(): void {
      this.showEditModal = false;
      this.selectedRole = null;
      this.validationErrors = {};
      this.editForm = {
        description: '',
        permissions: [],
      };
      this.selectedPermissionIds = [];
    },

    openDeleteModal(role: Role): void {
      this.showDeleteModal = true;
      this.selectedRole = role;
    },

    closeDeleteModal(): void {
      this.showDeleteModal = false;
      this.selectedRole = null;
    },

    openPermissionsModal(role: Role): void {
      this.showPermissionsModal = true;
      this.selectedRole = role;
      this.selectedPermissionIds = role.permissions ? role.permissions.map(p => p.id) : [];
    },

    closePermissionsModal(): void {
      this.showPermissionsModal = false;
      this.selectedRole = null;
      this.selectedPermissionIds = [];
    },

    togglePermission(permissionId: string): void {
      const index = this.selectedPermissionIds.indexOf(permissionId);
      if (index > -1) {
        this.selectedPermissionIds.splice(index, 1);
      } else {
        this.selectedPermissionIds.push(permissionId);
      }
    },

    async doCreateRole(): Promise<void> {
      this.validationErrors = {};

      if (!this.createForm.name) {
        this.validationErrors.name = 'Name is required';
      }

      if (Object.keys(this.validationErrors).length > 0) {
        return;
      }

      this.createForm.permissions = this.selectedPermissionIds;
      this.saving = true;

      try {
        await api.createRole(this.createForm);
        this.closeCreateModal();
        await this.loadRoles();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          if (err.code === 'ROLE_EXISTS') {
            this.validationErrors.name = err.message;
          } else {
            this.error = err.message;
          }
        } else {
          this.error = 'Failed to create role';
        }
      } finally {
        this.saving = false;
      }
    },

    async doUpdateRole(): Promise<void> {
      if (!this.selectedRole) {
        return;
      }

      this.validationErrors = {};
      this.editForm.permissions = this.selectedPermissionIds;
      this.saving = true;

      try {
        await api.updateRole(this.selectedRole.id, this.editForm);
        this.closeEditModal();
        await this.loadRoles();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to update role';
        }
      } finally {
        this.saving = false;
      }
    },

    async doDeleteRole(): Promise<void> {
      if (!this.selectedRole) {
        return;
      }

      this.deleting = true;

      try {
        await api.deleteRole(this.selectedRole.id);
        this.closeDeleteModal();
        await this.loadRoles();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to delete role';
        }
      } finally {
        this.deleting = false;
      }
    },
  };
}
