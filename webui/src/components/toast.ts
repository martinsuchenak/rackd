// Toast notification component for user feedback

interface ToastMessage {
  id: string;
  message: string;
  type: 'success' | 'error' | 'warning' | 'info';
  duration?: number;
}

function toast() {
  return {
    toasts: [] as ToastMessage[],

    show(message: string, type: 'success' | 'error' | 'warning' | 'info' = 'info', duration = 5000) {
      const id = Date.now().toString() + Math.random().toString(36).substr(2, 9);
      const toast: ToastMessage = { id, message, type, duration };

      this.toasts.push(toast);

      if (duration > 0) {
        setTimeout(() => {
          this.remove(id);
        }, duration);
      }

      return id;
    },

    success(message: string, duration = 5000) {
      return this.show(message, 'success', duration);
    },

    error(message: string, duration = 7000) {
      return this.show(message, 'error', duration);
    },

    warning(message: string, duration = 6000) {
      return this.show(message, 'warning', duration);
    },

    info(message: string, duration = 5000) {
      return this.show(message, 'info', duration);
    },

    remove(id: string) {
      const index = this.toasts.findIndex((t: ToastMessage) => t.id === id);
      if (index > -1) {
        this.toasts.splice(index, 1);
      }
    },

    clear() {
      this.toasts = [];
    },
  };
}

// Global toast function for non-Alpine contexts
let globalToastInstance: ReturnType<typeof toast> | null = null;

export function showPermissionDenied() {
  const message = "You don't have permission to perform this action";
  if (globalToastInstance) {
    globalToastInstance.error(message);
  } else {
    console.error(message);
  }
}

export function toastComponent() {
  const instance = toast();
  if (!globalToastInstance) {
    globalToastInstance = instance;
  }
  return instance;
}

export { toast };
