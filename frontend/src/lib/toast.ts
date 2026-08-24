import { toast } from 'sonner';

export function toastError(message: string): void {
  toast.error(message);
}

export function toastInfo(message: string): void {
  toast.info(message);
}

export function toastSuccess(message: string): void {
  toast.success(message);
}

export function toastWarning(message: string): void {
  toast.warning(message);
}
