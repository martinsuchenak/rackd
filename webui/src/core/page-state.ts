export type ValidationErrors = Record<string, string | undefined>;

export interface ListPageState<TItem, TModal extends string> {
  items: TItem[];
  selectedItem: TItem | null;
  modalType: '' | TModal;
  loading: boolean;
  saving: boolean;
  deleting: boolean;
  error: string;
  validationErrors: ValidationErrors;
}
