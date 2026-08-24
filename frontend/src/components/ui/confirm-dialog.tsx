'use client';

import { useState, useEffect, type ReactNode } from 'react';
import { AlertTriangle } from 'lucide-react';
import { ActionButton } from '@/components/ui/action-button';
import { OverlayShell } from '@/components/ui/overlay-shell';

interface ConfirmDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  description: string;
  confirmText?: string;
  confirmValue?: string;
  confirmDisabledReason?: string;
  variant?: 'destructive';
  loading?: boolean;
  // Extra content rendered below the description (e.g. a "force" checkbox).
  children?: ReactNode;
}

export function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title,
  description,
  confirmText = 'Delete',
  confirmValue,
  confirmDisabledReason,
  variant,
  loading,
  children,
}: ConfirmDialogProps) {
  const [inputValue, setInputValue] = useState('');

  // Reset input when dialog opens/closes
  useEffect(() => {
    if (!open) setInputValue('');
  }, [open]);

  if (!open) return null;

  const canConfirm = confirmValue ? inputValue === confirmValue : true;

  return (
    <OverlayShell onClose={onClose}>
      <div className="relative bg-card border border-border rounded-lg shadow-xl max-w-md w-full mx-4 animate-fade-in">
        <div className="p-6">
          <div className="flex items-start gap-3">
            {variant === 'destructive' && (
              <div className="flex-shrink-0 h-9 w-9 rounded-full bg-status-error/10 flex items-center justify-center">
                <AlertTriangle className="h-5 w-5 text-status-error" />
              </div>
            )}
            <div className="min-w-0 flex-1">
              <h3 className="text-base font-semibold text-foreground">{title}</h3>
              <p className="mt-1 text-sm text-muted-foreground">{description}</p>

              {confirmValue && (
                <div className="mt-4">
                  <label className="block text-xs text-muted-foreground mb-1.5">
                    Type <span className="font-mono font-medium text-foreground">{confirmValue}</span> to confirm
                  </label>
                  <input
                    type="text"
                    value={inputValue}
                    onChange={(e) => setInputValue(e.target.value)}
                    className="w-full h-8 px-3 rounded border border-border bg-background text-sm
                      focus:outline-none focus:ring-1 focus:ring-ring font-mono"
                    placeholder={confirmValue}
                    autoFocus
                  />
                </div>
              )}

              {children && <div className="mt-4">{children}</div>}
            </div>
          </div>
        </div>

        <div className="flex items-center justify-end gap-2 px-6 py-4 border-t border-border">
          <ActionButton
            onClick={onClose}
            disabled={loading}
            intent="ghost"
            size="sm"
          >
            Cancel
          </ActionButton>
          <ActionButton
            onClick={onConfirm}
            disabled={!canConfirm || loading || !!confirmDisabledReason}
            disabledReason={
              confirmDisabledReason ||
              (!canConfirm ? 'Type the confirmation value to continue' : undefined)
            }
            intent={variant === 'destructive' ? 'destructive' : 'primary'}
            loading={loading}
            loadingLabel={confirmText}
            size="sm"
          >
            {confirmText}
          </ActionButton>
        </div>
      </div>
    </OverlayShell>
  );
}
