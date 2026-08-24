'use client';

import { useId } from 'react';
import type { ReactNode } from 'react';
import { X } from 'lucide-react';
import { cn } from '@/lib/utils';
import { OverlayShell } from '@/components/ui/overlay-shell';

type ModalSize = 'sm' | 'md' | 'lg' | 'xl';

interface ModalShellProps {
  title: string;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
  subtitle?: ReactNode;
  headerActions?: ReactNode;
  titleIcon?: ReactNode;
  size?: ModalSize;
  bodyClassName?: string;
  footerClassName?: string;
  panelClassName?: string;
}

const sizeClass: Record<ModalSize, string> = {
  sm: 'max-w-md',
  md: 'max-w-lg',
  lg: 'max-w-2xl',
  xl: 'max-w-4xl',
};

export function ModalShell({
  title,
  onClose,
  children,
  footer,
  subtitle,
  headerActions,
  titleIcon,
  size = 'md',
  bodyClassName,
  footerClassName,
  panelClassName,
}: ModalShellProps) {
  const titleId = useId();

  return (
    <OverlayShell onClose={onClose}>
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        className={cn(
          'relative w-full mx-4 rounded-lg border border-border bg-card shadow-xl animate-fade-in max-h-[90vh] overflow-y-auto',
          sizeClass[size],
          panelClassName,
        )}
      >
        <div className="flex items-center justify-between px-6 py-4 border-b border-border">
          <div className="flex items-center gap-3 min-w-0">
            {titleIcon}
            <div className="min-w-0">
              <div className="flex items-center gap-2 min-w-0">
                <h2 id={titleId} className="text-base font-semibold text-foreground truncate">{title}</h2>
                {headerActions}
              </div>
              {subtitle && <div className="mt-1 text-xs text-muted-foreground">{subtitle}</div>}
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded text-muted-foreground hover:text-foreground hover:bg-accent"
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className={cn('p-6 space-y-4', bodyClassName)}>{children}</div>
        {footer && <div className={cn('px-6 py-4 border-t border-border', footerClassName)}>{footer}</div>}
      </div>
    </OverlayShell>
  );
}
