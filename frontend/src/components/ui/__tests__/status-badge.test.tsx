/**
 * Tests for the StatusBadge component.
 *
 * Covers rendering with different status values, label display, dot indicator,
 * and CSS class application.
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { StatusBadge } from '@/components/ui/status-badge';

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

describe('StatusBadge', () => {
  it('renders with "active" status', () => {
    render(<StatusBadge status="active" />);
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('renders with "error" status', () => {
    render(<StatusBadge status="error" />);
    expect(screen.getByText('Error')).toBeInTheDocument();
  });

  it('renders with "warning" status', () => {
    render(<StatusBadge status="warning" />);
    expect(screen.getByText('Warning')).toBeInTheDocument();
  });

  it('renders with "pending" status', () => {
    render(<StatusBadge status="pending" />);
    expect(screen.getByText('Pending')).toBeInTheDocument();
  });

  it('renders with "disconnected" status', () => {
    render(<StatusBadge status="disconnected" />);
    expect(screen.getByText('Disconnected')).toBeInTheDocument();
  });

  it('renders custom label when provided', () => {
    render(<StatusBadge status="active" label="Online" />);
    expect(screen.getByText('Online')).toBeInTheDocument();
  });

  it('capitalizes first letter of status as default label', () => {
    render(<StatusBadge status="running" />);
    expect(screen.getByText('Running')).toBeInTheDocument();
  });

  // ---------------------------------------------------------------------------
  // CSS classes
  // ---------------------------------------------------------------------------

  it('applies success background class for active status', () => {
    const { container } = render(<StatusBadge status="active" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain('bg-status-success');
  });

  it('applies error background class for error status', () => {
    const { container } = render(<StatusBadge status="error" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain('bg-status-error');
  });

  it('applies warning background class for warning status', () => {
    const { container } = render(<StatusBadge status="warning" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain('bg-status-warning');
  });

  it('applies info background class for pending status', () => {
    const { container } = render(<StatusBadge status="pending" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain('bg-status-info');
  });

  it('normalizes sync and drift-style statuses', () => {
    const { container } = render(<StatusBadge status="OutOfSync" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain('bg-status-warning');
  });

  it('applies permission denial status as error', () => {
    const { container } = render(<StatusBadge status="denied" label="Denied" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain('bg-status-error');
  });

  it('applies custom className', () => {
    const { container } = render(<StatusBadge status="active" className="my-custom-class" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain('my-custom-class');
  });

  // ---------------------------------------------------------------------------
  // Dot indicator
  // ---------------------------------------------------------------------------

  it('shows dot indicator by default', () => {
    const { container } = render(<StatusBadge status="active" />);
    const dots = container.querySelectorAll('.rounded-full');
    expect(dots.length).toBeGreaterThan(0);
  });

  it('hides dot indicator when showDot is false', () => {
    const { container } = render(<StatusBadge status="active" showDot={false} />);
    // Without the dot, there should be no nested span with rounded-full for the dot
    const badge = container.firstChild as HTMLElement;
    const nestedSpans = badge.querySelectorAll('span > span');
    // Should only be the text, no dot container
    expect(nestedSpans.length).toBe(0);
  });

  it('renders a custom icon instead of the dot indicator', () => {
    const { container } = render(
      <StatusBadge status="synced" icon={<svg data-testid="badge-icon" />} />
    );
    expect(screen.getByTestId('badge-icon')).toBeInTheDocument();
    const dots = container.querySelectorAll('.animate-pulse-dot');
    expect(dots.length).toBe(0);
  });

  // ---------------------------------------------------------------------------
  // Size variants
  // ---------------------------------------------------------------------------

  it('renders with small size', () => {
    const { container } = render(<StatusBadge status="active" size="sm" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain('text-[10px]');
  });

  it('renders with medium size (default)', () => {
    const { container } = render(<StatusBadge status="active" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain('text-xs');
  });

  it('renders with large size', () => {
    const { container } = render(<StatusBadge status="active" size="lg" />);
    const badge = container.firstChild as HTMLElement;
    expect(badge.className).toContain('text-sm');
  });

  // ---------------------------------------------------------------------------
  // Pulse behavior
  // ---------------------------------------------------------------------------

  it('shows pulse animation for active statuses', () => {
    const { container } = render(<StatusBadge status="active" />);
    const pulseElement = container.querySelector('.animate-pulse-dot');
    expect(pulseElement).not.toBeNull();
  });

  it('shows pulse when explicitly set via pulse prop', () => {
    const { container } = render(<StatusBadge status="error" pulse={true} />);
    const pulseElement = container.querySelector('.animate-pulse-dot');
    expect(pulseElement).not.toBeNull();
  });
});
