'use client';

import type { ReactNode } from 'react';
import { cn, statusBgColor, statusDotColor } from '@/lib/utils';
import { cva, type VariantProps } from 'class-variance-authority';

const statusBadgeVariants = cva(
  'inline-flex items-center gap-1.5 rounded-full font-medium transition-colors',
  {
    variants: {
      size: {
        sm: 'px-2 py-0.5 text-[10px] leading-4',
        md: 'px-2.5 py-0.5 text-xs leading-5',
        lg: 'px-3 py-1 text-sm leading-5',
      },
    },
    defaultVariants: {
      size: 'md',
    },
  }
);

interface StatusBadgeProps extends VariantProps<typeof statusBadgeVariants> {
  status: string;
  label?: string;
  icon?: ReactNode;
  showDot?: boolean;
  pulse?: boolean;
  className?: string;
}

export function StatusBadge({
  status,
  label,
  icon,
  showDot = true,
  pulse = false,
  size,
  className,
}: StatusBadgeProps) {
  const displayLabel = label || status.charAt(0).toUpperCase() + status.slice(1).replace(/([A-Z])/g, ' $1');
  const isActive = ['active', 'healthy', 'running', 'synced', 'connected'].includes(
    status.toLowerCase()
  );

  return (
    <span className={cn(statusBadgeVariants({ size }), statusBgColor(status), className)}>
      {icon ? <span className="inline-flex h-3 w-3 shrink-0 items-center justify-center">{icon}</span> : null}
      {!icon && showDot && (
        <span className="relative flex h-1.5 w-1.5">
          {(pulse || isActive) && (
            <span
              className={cn(
                'absolute inline-flex h-full w-full rounded-full opacity-75 animate-pulse-dot',
                statusDotColor(status)
              )}
            />
          )}
          <span
            className={cn('relative inline-flex rounded-full h-1.5 w-1.5', statusDotColor(status))}
          />
        </span>
      )}
      {displayLabel}
    </span>
  );
}
