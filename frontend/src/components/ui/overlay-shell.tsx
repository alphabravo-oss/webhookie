'use client';

import { useEffect, useRef, type ReactNode } from 'react';
import { cn } from '@/lib/utils';

type OverlayPlacement = 'center' | 'right';

interface OverlayShellProps {
  onClose: () => void;
  children: ReactNode;
  placement?: OverlayPlacement;
  rootClassName?: string;
  backdropClassName?: string;
  closeOnBackdrop?: boolean;
}

const placementClass: Record<OverlayPlacement, string> = {
  center: 'items-center justify-center',
  right: 'justify-end',
};

const focusableSelector = [
  'a[href]',
  'button:not([disabled])',
  'textarea:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',');

function getFocusable(container: HTMLElement) {
  return Array.from(container.querySelectorAll<HTMLElement>(focusableSelector)).filter(
    (element) => !element.getAttribute('aria-hidden'),
  );
}

export function OverlayShell({
  onClose,
  children,
  placement = 'center',
  rootClassName,
  backdropClassName,
  closeOnBackdrop = true,
}: OverlayShellProps) {
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const previousActive = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const root = rootRef.current;
    if (!root) return;

    const focusTarget = getFocusable(root)[0] ?? root;
    focusTarget.focus({ preventScroll: true });

    return () => {
      if (previousActive && document.contains(previousActive)) {
        previousActive.focus({ preventScroll: true });
      }
    };
  }, []);

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        onClose();
        return;
      }
      if (e.key !== 'Tab') return;

      const root = rootRef.current;
      if (!root) return;

      const focusable = getFocusable(root);
      if (focusable.length === 0) {
        e.preventDefault();
        root.focus({ preventScroll: true });
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const active = document.activeElement;

      if (e.shiftKey && (!active || active === first || !root.contains(active))) {
        e.preventDefault();
        last.focus({ preventScroll: true });
        return;
      }

      if (!e.shiftKey && active === last) {
        e.preventDefault();
        first.focus({ preventScroll: true });
      }
    }
    document.addEventListener('keydown', handleKey);
    return () => document.removeEventListener('keydown', handleKey);
  }, [onClose]);

  return (
    <div
      ref={rootRef}
      tabIndex={-1}
      className={cn('fixed inset-0 z-50 flex', placementClass[placement], rootClassName)}
    >
      <button
        type="button"
        aria-label="Close overlay"
        aria-hidden="true"
        tabIndex={-1}
        className={cn('absolute inset-0 border-0 bg-black/50 p-0 backdrop-blur-sm', backdropClassName)}
        onClick={closeOnBackdrop ? onClose : undefined}
      />
      {children}
    </div>
  );
}
