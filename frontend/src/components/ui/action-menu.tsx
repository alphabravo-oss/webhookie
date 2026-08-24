'use client';

import { useState, useRef, useEffect, useLayoutEffect } from 'react';
import { createPortal } from 'react-dom';
import { MoreHorizontal } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface ActionMenuItem {
  label: string;
  icon?: React.ReactNode;
  onClick: () => void;
  variant?: 'default' | 'destructive';
  disabled?: boolean;
  disabledReason?: string;
  separator?: boolean;
}

interface ActionMenuProps {
  items: ActionMenuItem[];
}

// 208px (w-52) gives multi-word labels like "Registration Command" room to sit
// on a single line at text-xs; the previous w-44 (176px) was just narrow enough
// to wrap, which looked broken next to the single-line items.
const MENU_WIDTH = 208;
const MENU_GAP = 4;
const MENU_MIN_HEIGHT = 120;

// The menu used to render inline as `position: absolute`, which any
// `overflow-hidden`/`overflow-x-auto` ancestor (every DataTable wrapper, the
// dashboard main pane, etc.) clipped. We now portal the menu to <body> and
// position it with `position: fixed` from the trigger button's bounding rect,
// which dodges every parent overflow boundary cleanly.
export function ActionMenu({ items }: ActionMenuProps) {
  const [open, setOpen] = useState(false);
  const [coords, setCoords] = useState<{ top: number; left: number } | null>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      const target = e.target as Node;
      if (menuRef.current?.contains(target) || buttonRef.current?.contains(target)) return;
      setOpen(false);
    }
    function handleClose() {
      setOpen(false);
    }
    document.addEventListener('mousedown', handleClick);
    // Close when the page scrolls or resizes so the menu doesn't drift away
    // from its trigger; cheaper than recomputing coords on every scroll tick.
    window.addEventListener('scroll', handleClose, true);
    window.addEventListener('resize', handleClose);
    return () => {
      document.removeEventListener('mousedown', handleClick);
      window.removeEventListener('scroll', handleClose, true);
      window.removeEventListener('resize', handleClose);
    };
  }, [open]);

  // Recompute position after open so we land relative to the button.
  useLayoutEffect(() => {
    if (!open || !buttonRef.current) return;
    const rect = buttonRef.current.getBoundingClientRect();
    const spaceBelow = window.innerHeight - rect.bottom;
    const flipUp = spaceBelow < MENU_MIN_HEIGHT;
    const top = flipUp ? rect.top - MENU_GAP : rect.bottom + MENU_GAP;
    // Right-align under the button: the menu's right edge lines up with the
    // button's right edge. Clamp to the viewport so it doesn't go offscreen
    // on narrow widths.
    let left = rect.right - MENU_WIDTH;
    if (left < 8) left = 8;
    if (left + MENU_WIDTH > window.innerWidth - 8) left = window.innerWidth - MENU_WIDTH - 8;
    setCoords({ top, left });
  }, [open]);

  const handleToggle = (e: React.MouseEvent) => {
    e.stopPropagation();
    setOpen(!open);
  };

  return (
    <>
      <button
        ref={buttonRef}
        onClick={handleToggle}
        className="inline-flex items-center justify-center h-7 w-7 rounded
          text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
      >
        <MoreHorizontal className="h-4 w-4" />
      </button>

      {open && coords && typeof document !== 'undefined' && createPortal(
        <div
          ref={menuRef}
          style={{
            position: 'fixed',
            top: coords.top,
            left: coords.left,
            width: MENU_WIDTH,
            transform: coords.top < (buttonRef.current?.getBoundingClientRect().top ?? 0)
              ? 'translateY(-100%)'
              : undefined,
          }}
          className="rounded-md border border-border bg-popover p-1 shadow-lg z-[9999]"
          onClick={(e) => e.stopPropagation()}
        >
          {items.map((item, i) => (
            <div key={i}>
              {item.separator && i > 0 && (
                <div className="my-1 h-px bg-border" />
              )}
              <button
                onClick={() => {
                  if (!item.disabled) {
                    item.onClick();
                    setOpen(false);
                  }
                }}
                disabled={item.disabled}
                title={item.disabledReason}
                className={cn(
                  'w-full flex items-center gap-2 px-2.5 py-1.5 rounded text-xs transition-colors whitespace-nowrap',
                  item.disabled && 'opacity-50 cursor-not-allowed',
                  item.variant === 'destructive'
                    ? 'text-status-error hover:bg-status-error/10'
                    : 'text-popover-foreground hover:bg-accent',
                )}
              >
                {item.icon && <span className="flex-shrink-0">{item.icon}</span>}
                {item.label}
              </button>
            </div>
          ))}
        </div>,
        document.body,
      )}
    </>
  );
}
