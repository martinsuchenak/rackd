export type ModalSize = 'md' | 'lg' | '2xl' | '4xl';

const modalSizeClasses: Record<ModalSize, string> = {
  md: 'max-w-md',
  lg: 'max-w-lg',
  '2xl': 'max-w-2xl',
  '4xl': 'max-w-4xl',
};

export interface UIStore {
  modalViewport(): string;
  modalBackdrop(): string;
  modalPanel(size?: ModalSize, scrollable?: boolean): string;
  modalCloseButton(): string;
}

export function createUIStore(): UIStore {
  return {
    modalViewport(): string {
      return 'fixed inset-0 z-50 flex items-center justify-center p-4';
    },

    modalBackdrop(): string {
      return 'fixed inset-0 bg-black/50';
    },

    modalPanel(size: ModalSize = 'lg', scrollable = false): string {
      const base = [
        'relative',
        'bg-white',
        'dark:bg-gray-800',
        'rounded-lg',
        'shadow-xl',
        modalSizeClasses[size],
        'w-full',
        'p-6',
      ];

      if (scrollable) {
        base.push('max-h-[90vh]', 'overflow-y-auto');
      }

      return base.join(' ');
    },

    modalCloseButton(): string {
      return 'absolute top-4 right-4 text-gray-600 hover:text-gray-800 dark:text-gray-300 dark:hover:text-gray-100 focus:outline-none focus:ring-[3px] focus:ring-blue-500 rounded-full p-2 cursor-pointer transition-colors min-w-[44px] min-h-[44px] flex items-center justify-center';
    },
  };
}
