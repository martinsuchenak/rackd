// Roles Component for Rackd Web UI

import type { Role, Permission, RoleFilter, CreateRoleRequest, UpdateRoleRequest } from '../core/types';
import { api, RackdAPIError } from '../core/api';
import type { ListPageState } from '../core/page-state';

type ModalType = '' | 'create' | 'edit' | 'delete' | 'view' | 'permissions';

interface RolesListData extends ListPageState<Role, Exclude<ModalType, ''>> {
  roles: Role[];
  permissions: Permission[];
  filter: RoleFilter;
  page: number;
  pageSize: number;
  totalPages: number;
  pagedRoles: Role[];
  modalType: ModalType;
  selectedRole: Role | null;
  get items(): Role[];
  get selectedItem(): Role | null;
  get showCreateModal(): boolean;
  get showEditModal(): boolean;
  get showDeleteModal(): boolean;
  get showViewModal(): boolean;
  get showPermissionsModal(): boolean;
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
  openViewModal(role: Role): void;
  closeViewModal(): void;
  openPermissionsModal(role: Role): void;
  closePermissionsModal(): void;
  closeModal(): void;
  doCreateRole(): Promise<void>;
  doUpdateRole(): Promise<void>;
  doDeleteRole(): Promise<void>;
  save(): Promise<void>;
  deleteConfirmed(): Promise<void>;
  togglePermission(permissionId: string): void;
  getSelectedRoleName(): string;
  getSelectedRoleDescription(): string;
  getSelectedRolePermissionsCount(): number;
  hasSelectedRoleDescription(): boolean;
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
    modalType: '' as ModalType,
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

    get items(): Role[] { return this.roles; },
    get selectedItem(): Role | null { return this.selectedRole; },
    get showCreateModal(): boolean { return this.modalType === 'create'; },
    get showEditModal(): boolean { return this.modalType === 'edit'; },
    get showDeleteModal(): boolean { return this.modalType === 'delete'; },
    get showViewModal(): boolean { return this.modalType === 'view'; },
    get showPermissionsModal(): boolean { return this.modalType === 'permissions'; },

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

    get groupedPermissions(): Record<string, Permission[]> {
      if (!this.selectedRole?.permissions) return {};
      return this.selectedRole.permissions.reduce((acc, perm) => {
        if (!acc[perm.resource]) acc[perm.resource] = [];
        acc[perm.resource].push(perm);
        return acc;
      }, {} as Record<string, Permission[]>);
    },

    formatDate(date: string | Date): string {
      if (!date) return '-';
      const d = typeof date === 'string' ? new Date(date) : date;
      return d.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
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
      this.modalType = '';
      this.selectedRole = null;
      this.validationErrors = {};
      this.createForm = {
        name: '',
        description: '',
        permissions: [],
      };
      this.selectedPermissionIds = [];
      this.modalType = 'create';
    },

    closeCreateModal(): void {
      this.closeModal();
    },

    openEditModal(role: Role): void {
      this.modalType = '';
      this.selectedRole = role;
      this.validationErrors = {};
      this.editForm = {
        description: role.description || '',
        permissions: role.permissions ? role.permissions.map(p => p.id) : [],
      };
      this.selectedPermissionIds = role.permissions ? role.permissions.map(p => p.id) : [];
      this.modalType = 'edit';
    },

    closeEditModal(): void {
      this.closeModal();
    },

    openDeleteModal(role: Role): void {
      this.modalType = '';
      this.selectedRole = role;
      this.modalType = 'delete';
    },

    closeDeleteModal(): void {
      this.closeModal();
    },

    openViewModal(role: Role): void {
      this.modalType = '';
      this.selectedRole = role;
      this.modalType = 'view';
    },

    closeViewModal(): void {
      this.closeModal();
    },

    openPermissionsModal(role: Role): void {
      this.modalType = '';
      this.selectedRole = role;
      this.selectedPermissionIds = role.permissions ? role.permissions.map(p => p.id) : [];
      this.modalType = 'permissions';
    },

    closePermissionsModal(): void {
      this.closeModal();
    },

    closeModal(): void {
      const previousModalType = this.modalType;
      this.modalType = '';
      this.selectedRole = null;
      this.validationErrors = {};

      if (previousModalType === 'create') {
        this.createForm = {
          name: '',
          description: '',
          permissions: [],
        };
      }

      if (previousModalType === 'edit') {
        this.editForm = {
          description: '',
          permissions: [],
        };
      }

      if (previousModalType === 'create' || previousModalType === 'edit' || previousModalType === 'permissions') {
        this.selectedPermissionIds = [];
      }
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

    async save(): Promise<void> {
      if (this.modalType === 'edit') {
        await this.doUpdateRole();
        return;
      }
      await this.doCreateRole();
    },

    async deleteConfirmed(): Promise<void> {
      await this.doDeleteRole();
    },

    getSelectedRoleName(): string {
      return this.selectedRole ? this.selectedRole.name : '';
    },

    getSelectedRoleDescription(): string {
      return this.selectedRole?.description || '';
    },

    getSelectedRolePermissionsCount(): number {
      return this.selectedRole?.permissions ? this.selectedRole.permissions.length : 0;
    },

    hasSelectedRoleDescription(): boolean {
      return !!this.selectedRole?.description;
    }
  };
}
