// Users Component for Rackd Web UI

import type { User, UserFilter, CreateUserRequest, UpdateUserRequest, Role } from '../core/types';
import { api, RackdAPIError } from '../core/api';
import { formatDate } from '../core/utils';

interface UsersListData {
  users: User[];
  loading: boolean;
  error: string;
  filter: UserFilter;
  page: number;
  pageSize: number;
  totalPages: number;
  pagedUsers: User[];
  showCreateModal: boolean;
  showEditModal: boolean;
  showDeleteModal: boolean;
  showPasswordModal: boolean;
  showResetPasswordModal: boolean;
  selectedUser: User | null;
  saving: boolean;
  deleting: boolean;
  validationErrors: Record<string, string>;
  createForm: CreateUserRequest;
  editForm: { email: string; full_name: string; is_active: boolean };
  passwordForm: { old_password: string; new_password: string; confirm_password: string };
  resetPasswordForm: { new_password: string; confirm_password: string };
  currentUser: User | null;
  availableRoles: Role[];
  userRolesCache: Map<string, Role[]>;
  showRoleManager: boolean;
  roleSaving: boolean;
  init(): Promise<void>;
  loadUsers(): Promise<void>;
  loadAvailableRoles(): Promise<void>;
  loadUserRoles(userId: string): Promise<Role[]>;
  applyFilter(): void;
  clearFilter(): void;
  goToPage(p: number): void;
  openCreateModal(): void;
  closeCreateModal(): void;
  openEditModal(user: User): void;
  closeEditModal(): void;
  openDeleteModal(user: User): void;
  closeDeleteModal(): void;
  openPasswordModal(user: User): void;
  closePasswordModal(): void;
  openResetPasswordModal(user: User): void;
  closeResetPasswordModal(): void;
  openRoleManager(user: User): void;
  closeRoleManager(): void;
  doCreateUser(): Promise<void>;
  doUpdateUser(): Promise<void>;
  doDeleteUser(): Promise<void>;
  doChangePassword(): Promise<void>;
  doResetPassword(): Promise<void>;
  grantRole(role: Role): Promise<void>;
  revokeRole(role: Role): Promise<void>;
  logout(): void;
  getSelectedUsername(): string;
  canResetPassword(user: User): boolean;
  canDeleteUser(user: User): boolean;
  hasUserRoles(user: User): boolean;
  getUserRoles(user: User): Role[];
}

export function usersList() {
  return {
    users: [] as User[],
    loading: true,
    error: '',
    filter: {} as UserFilter,
    page: 1,
    pageSize: 10,
    showCreateModal: false,
    showEditModal: false,
    showDeleteModal: false,
    showPasswordModal: false,
    showResetPasswordModal: false,
    selectedUser: null as User | null,
    saving: false,
    deleting: false,
    validationErrors: {} as Record<string, string>,
    createForm: {
      username: '',
      password: '',
      email: '',
      full_name: '',
      is_admin: false,
    } as CreateUserRequest,
    editForm: {
      email: '',
      full_name: '',
      is_active: true,
    },
    passwordForm: {
      old_password: '',
      new_password: '',
      confirm_password: '',
    },
    resetPasswordForm: {
      new_password: '',
      confirm_password: '',
    },
    currentUser: null as User | null,
    availableRoles: [] as Role[],
    userRolesCache: new Map<string, Role[]>(),
    showRoleManager: false,
    roleSaving: false,

    get totalPages(): number {
      return Math.ceil(this.users.length / this.pageSize) || 1;
    },

    get pagedUsers(): User[] {
      const start = (this.page - 1) * this.pageSize;
      return this.users.slice(start, start + this.pageSize);
    },

    get deleteModalTitle(): string {
      return 'Delete User';
    },

    get deleteModalName(): string {
      return this.selectedUser?.username || '';
    },

    async init(): Promise<void> {
      await this.loadUsers();
      await this.loadCurrentUser();
    },

    async loadUsers(): Promise<void> {
      this.loading = true;
      this.error = '';

      try {
        this.users = await api.listUsers(this.filter);
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to load users';
        }
      } finally {
        this.loading = false;
      }
    },

    async loadCurrentUser(): Promise<void> {
      try {
        this.currentUser = await api.getCurrentUser();
      } catch {
        this.currentUser = null;
      }
      await this.loadAvailableRoles();
    },

    async loadAvailableRoles(): Promise<void> {
      try {
        this.availableRoles = await api.listRoles({});
      } catch (err) {
        console.error('Failed to load roles:', err);
        if (err instanceof RackdAPIError) {
          this.error = 'Failed to load roles: ' + err.message;
        }
        this.availableRoles = [];
      }
    },

    async loadUserRoles(userId: string): Promise<Role[]> {
      if (this.userRolesCache.has(userId)) {
        return this.userRolesCache.get(userId)!;
      }

      try {
        const roles = await api.getUserRoles(userId);
        this.userRolesCache.set(userId, roles);
        return roles;
      } catch {
        return [];
      }
    },

    applyFilter(): void {

      this.page = 1;
      this.loadUsers();
    },

    clearFilter(): void {
      this.filter = {} as UserFilter;
      this.page = 1;
      this.loadUsers();
    },

    goToPage(p: number): void {
      this.page = p;
    },

    openCreateModal(): void {
      this.showCreateModal = true;
      this.validationErrors = {};
      this.createForm = {
        username: '',
        password: '',
        email: '',
        full_name: '',
        is_admin: false,
      };
    },

    closeCreateModal(): void {
      this.showCreateModal = false;
      this.validationErrors = {};
      this.createForm = {
        username: '',
        password: '',
        email: '',
        full_name: '',
        is_admin: false,
      };
    },

    openEditModal(user: User): void {
      this.showEditModal = true;
      this.selectedUser = user;
      this.validationErrors = {};
      // Pre-populate edit form with user's current values
      this.editForm = {
        email: user.email || '',
        full_name: user.full_name || '',
        is_active: user.is_active !== false,
      };
    },

    closeEditModal(): void {
      this.showEditModal = false;
      this.selectedUser = null;
      this.validationErrors = {};
    },

    openDeleteModal(user: User): void {
      this.showDeleteModal = true;
      this.selectedUser = user;
    },

    closeDeleteModal(): void {
      this.showDeleteModal = false;
      this.selectedUser = null;
    },

    openPasswordModal(user: User): void {
      this.showPasswordModal = true;
      this.selectedUser = user;
      this.validationErrors = {};
      this.passwordForm = {
        old_password: '',
        new_password: '',
        confirm_password: '',
      };
    },

    closePasswordModal(): void {
      this.showPasswordModal = false;
      this.selectedUser = null;
      this.validationErrors = {};
      this.passwordForm = {
        old_password: '',
        new_password: '',
        confirm_password: '',
      };
    },

    openResetPasswordModal(user: User): void {
      this.showResetPasswordModal = true;
      this.selectedUser = user;
      this.validationErrors = {};
      this.resetPasswordForm = {
        new_password: '',
        confirm_password: '',
      };
    },

    closeResetPasswordModal(): void {
      this.showResetPasswordModal = false;
      this.selectedUser = null;
      this.validationErrors = {};
      this.resetPasswordForm = {
        new_password: '',
        confirm_password: '',
      };
    },

    hasRole(role: Role): boolean {
      if (!this.selectedUser?.roles) {
        return false;
      }
      return this.selectedUser.roles.some((r: Role) => r.id === role.id);
    },

    async openRoleManager(user: User): Promise<void> {
      this.selectedUser = user;
      // Load roles and assign to selectedUser for hasRole() to work
      this.selectedUser.roles = await this.loadUserRoles(user.id);
      this.showRoleManager = true;
    },

    closeRoleManager(): void {
      this.showRoleManager = false;
      this.selectedUser = null;
    },

    async grantRole(role: Role): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      this.roleSaving = true;

      try {
        await api.grantRole(this.selectedUser.id, role.id);
        this.userRolesCache.delete(this.selectedUser.id);
        this.selectedUser.roles = await this.loadUserRoles(this.selectedUser.id);
        await this.loadUsers();

        // Refresh permissions if user is modifying their own roles
        if (this.currentUser && this.currentUser.id === this.selectedUser.id) {
          window.dispatchEvent(new Event('permissions:refresh'));
        }
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to grant role';
        }
      } finally {
        this.roleSaving = false;
      }
    },

    async revokeRole(role: Role): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      this.roleSaving = true;

      try {
        await api.revokeRole(this.selectedUser.id, role.id);
        this.userRolesCache.delete(this.selectedUser.id);
        this.selectedUser.roles = await this.loadUserRoles(this.selectedUser.id);
        await this.loadUsers();

        // Refresh permissions if user is modifying their own roles
        if (this.currentUser && this.currentUser.id === this.selectedUser.id) {
          window.dispatchEvent(new Event('permissions:refresh'));
        }
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to revoke role';
        }
      } finally {
        this.roleSaving = false;
      }
    },

    async doCreateUser(): Promise<void> {
      this.validationErrors = {};

      if (!this.createForm.username) {
        this.validationErrors.username = 'Username is required';
      }

      if (!this.createForm.password) {
        this.validationErrors.password = 'Password is required';
      } else if (this.createForm.password.length < 8) {
        this.validationErrors.password = 'Password must be at least 8 characters';
      }

      if (!this.createForm.email) {
        this.validationErrors.email = 'Email is required';
      } else if (!this.createForm.email.includes('@')) {
        this.validationErrors.email = 'Invalid email format';
      }

      if (Object.keys(this.validationErrors).length > 0) {
        return;
      }

      this.saving = true;

      try {
        await api.createUser(this.createForm);
        this.closeCreateModal();
        await this.loadUsers();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          if (err.code === 'USERNAME_EXISTS') {
            this.validationErrors.username = err.message;
          } else if (err.code === 'EMAIL_EXISTS') {
            this.validationErrors.email = err.message;
          } else {
            this.error = err.message;
          }
        } else {
          this.error = 'Failed to create user';
        }
      } finally {
        this.saving = false;
      }
    },

    async doUpdateUser(): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      this.validationErrors = {};
      const updates: UpdateUserRequest = {};

      if (this.editForm.email) {
        if (!this.editForm.email.includes('@')) {
          this.validationErrors.email = 'Invalid email format';
          return;
        }
        updates.email = this.editForm.email;
      }

      if (this.editForm.full_name) {
        updates.full_name = this.editForm.full_name;
      }

      updates.is_active = this.editForm.is_active;

      if (Object.keys(this.validationErrors).length > 0) {
        return;
      }

      this.saving = true;

      try {
        await api.updateUser(this.selectedUser.id, updates);
        this.closeEditModal();
        await this.loadUsers();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          if (err.code === 'EMAIL_EXISTS') {
            this.validationErrors.email = err.message;
          } else {
            this.error = err.message;
          }
        } else {
          this.error = 'Failed to update user';
        }
      } finally {
        this.saving = false;
      }
    },

    async doDeleteUser(): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      if (this.currentUser && this.currentUser.id === this.selectedUser.id) {
        this.error = 'Cannot delete your own account';
        return;
      }

      this.deleting = true;

      try {
        await api.deleteUser(this.selectedUser.id);
        this.closeDeleteModal();
        await this.loadUsers();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to delete user';
        }
      } finally {
        this.deleting = false;
      }
    },

    async doChangePassword(): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      this.validationErrors = {};

      if (!this.passwordForm.old_password) {
        this.validationErrors.old_password = 'Old password is required';
      }

      if (!this.passwordForm.new_password) {
        this.validationErrors.new_password = 'New password is required';
      } else if (this.passwordForm.new_password.length < 8) {
        this.validationErrors.new_password = 'Password must be at least 8 characters';
      }

      if (this.passwordForm.new_password !== this.passwordForm.confirm_password) {
        this.validationErrors.confirm_password = 'Passwords do not match';
      }

      if (Object.keys(this.validationErrors).length > 0) {
        return;
      }

      this.saving = true;

      try {
        await api.changePassword(this.selectedUser.id, {
          old_password: this.passwordForm.old_password,
          new_password: this.passwordForm.new_password,
        });
        this.closePasswordModal();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          if (err.code === 'INVALID_PASSWORD') {
            this.validationErrors.old_password = err.message;
          } else {
            this.error = err.message;
          }
        } else {
          this.error = 'Failed to change password';
        }
      } finally {
        this.saving = false;
      }
    },

    async doResetPassword(): Promise<void> {
      if (!this.selectedUser) {
        return;
      }

      this.validationErrors = {};

      if (!this.resetPasswordForm.new_password) {
        this.validationErrors.new_password = 'New password is required';
      } else if (this.resetPasswordForm.new_password.length < 8) {
        this.validationErrors.new_password = 'Password must be at least 8 characters';
      }

      if (this.resetPasswordForm.new_password !== this.resetPasswordForm.confirm_password) {
        this.validationErrors.confirm_password = 'Passwords do not match';
      }

      if (Object.keys(this.validationErrors).length > 0) {
        return;
      }

      this.saving = true;

      try {
        await api.resetPassword(this.selectedUser.id, this.resetPasswordForm.new_password);
        this.closeResetPasswordModal();
      } catch (err) {
        if (err instanceof RackdAPIError) {
          this.error = err.message;
        } else {
          this.error = 'Failed to reset password';
        }
      } finally {
        this.saving = false;
      }
    },

    async logout(): Promise<void> {
      try {
        await api.logout();
      } catch {
        // Continue with redirect even if server call fails
      }
      window.location.href = '/login';
    },

    formatDate: (dateString: string) => {
      return formatDate(dateString);
    },

    getUserInitial(username: string): string {
      return username ? username.charAt(0).toUpperCase() : '';
    },

    getSelectedUsername(): string {
      return this.selectedUser?.username || '';
    },

    canResetPassword(user: User): boolean {
      return !!(this.currentUser && this.currentUser.id !== user.id);
    },

    canDeleteUser(user: User): boolean {
      return !!(this.currentUser && this.currentUser.id !== user.id);
    },

    hasUserRoles(user: User | null): boolean {
      return !!(user && user.roles && user.roles.length > 0);
    },

    getUserRoles(user: User | null): Role[] {
      return user?.roles || [];
    },

    canUpdateRoles(): boolean {
      // @ts-ignore - Alpine store access
      return Alpine.store('permissions').canUpdate('roles');
    },

    canListRoles(): boolean {
      // @ts-ignore - Alpine store access
      return Alpine.store('permissions').canList('roles');
    },

    getLastLoginLabel(user: User): string {
      return user.last_login_at ? this.formatDate(user.last_login_at) : 'Never';
    },

    getSelectedUserRolesCount(): number {
      return this.selectedUser?.roles?.length || 0;
    }
  };
}

