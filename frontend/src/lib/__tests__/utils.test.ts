/**
 * Tests for src/lib/utils.ts
 *
 * Covers cn(), formatBytes(), formatCPU(), statusColor(), and other utility
 * functions exported from the utils module.
 */

import {
  cn,
  formatBytes,
  formatCPU,
  formatPercentage,
  statusColor,
  statusBgColor,
  statusDotColor,
  providerDisplayName,
  truncate,
  stringToColor,
  gaugeColor,
  gaugeTextColor,
} from '@/lib/utils';

// ---------------------------------------------------------------------------
// cn()
// ---------------------------------------------------------------------------

describe('cn()', () => {
  it('merges class names correctly', () => {
    const result = cn('foo', 'bar');
    expect(result).toContain('foo');
    expect(result).toContain('bar');
  });

  it('handles conditional classes', () => {
    const hide: boolean = false;
    const result = cn('base', hide && 'hidden', 'visible');
    expect(result).toContain('base');
    expect(result).toContain('visible');
    expect(result).not.toContain('hidden');
  });

  it('handles undefined and null values', () => {
    const result = cn('base', undefined, null, 'end');
    expect(result).toContain('base');
    expect(result).toContain('end');
  });

  it('resolves tailwind conflicts (last wins)', () => {
    // twMerge should keep the last conflicting utility
    const result = cn('p-4', 'p-2');
    expect(result).toBe('p-2');
  });

  it('returns empty string for no arguments', () => {
    const result = cn();
    expect(result).toBe('');
  });

  it('handles arrays of class names', () => {
    const result = cn(['foo', 'bar']);
    expect(result).toContain('foo');
    expect(result).toContain('bar');
  });
});

// ---------------------------------------------------------------------------
// formatBytes()
// ---------------------------------------------------------------------------

describe('formatBytes()', () => {
  it('formats 0 bytes', () => {
    expect(formatBytes(0)).toBe('0 B');
  });

  it('formats bytes (< 1 KiB)', () => {
    expect(formatBytes(512)).toBe('512 B');
  });

  it('formats 1024 bytes as 1 KiB', () => {
    expect(formatBytes(1024)).toBe('1 KiB');
  });

  it('formats 1048576 bytes as 1 MiB', () => {
    expect(formatBytes(1048576)).toBe('1 MiB');
  });

  it('formats 1073741824 bytes as 1 GiB', () => {
    expect(formatBytes(1073741824)).toBe('1 GiB');
  });

  it('formats with custom decimals', () => {
    expect(formatBytes(1536, 2)).toBe('1.5 KiB');
  });

  it('formats large values (TiB range)', () => {
    const tib = 1024 ** 4;
    expect(formatBytes(tib)).toBe('1 TiB');
  });
});

// ---------------------------------------------------------------------------
// formatCPU()
// ---------------------------------------------------------------------------

describe('formatCPU()', () => {
  it('formats millicores below 1000', () => {
    expect(formatCPU(250)).toBe('250m');
  });

  it('formats exactly 1000 millicores as cores (trailing zero stripped)', () => {
    expect(formatCPU(1000)).toBe('1 cores');
  });

  it('formats values above 1000 as cores', () => {
    expect(formatCPU(2500)).toBe('2.5 cores');
  });

  it('formats 0 millicores', () => {
    expect(formatCPU(0)).toBe('0m');
  });

  it('formats 500 millicores', () => {
    expect(formatCPU(500)).toBe('500m');
  });

  it('formats 4000 millicores as 4 cores (trailing zero stripped)', () => {
    expect(formatCPU(4000)).toBe('4 cores');
  });

  it('formats 2500 millicores as 2.5 cores', () => {
    expect(formatCPU(2500)).toBe('2.5 cores');
  });

  // Regression: prometheus rate() queries can return 118.99999999999999
  // for what should display as ~119m. The unrounded float used to bleed
  // straight to the UI as "118.99999999999999m"; the rounder pins it.
  it('rounds float-noisy millicores to an integer', () => {
    expect(formatCPU(118.99999999999999)).toBe('119m');
    expect(formatCPU(42.0000000001)).toBe('42m');
  });

  it('returns em-dash for null/undefined/NaN', () => {
    // @ts-expect-error testing runtime guard
    expect(formatCPU(null)).toBe('—');
    // @ts-expect-error testing runtime guard
    expect(formatCPU(undefined)).toBe('—');
    expect(formatCPU(NaN)).toBe('—');
  });
});

// ---------------------------------------------------------------------------
// formatPercentage()
// ---------------------------------------------------------------------------

describe('formatPercentage()', () => {
  it('formats percentage with default decimals', () => {
    expect(formatPercentage(45.678)).toBe('45.7%');
  });

  it('formats percentage with custom decimals', () => {
    expect(formatPercentage(45.678, 2)).toBe('45.68%');
  });

  it('formats zero percentage without trailing zero', () => {
    expect(formatPercentage(0)).toBe('0%');
  });

  it('formats 100 percentage without trailing zero', () => {
    expect(formatPercentage(100)).toBe('100%');
  });

  it('returns em-dash for null (no data)', () => {
    expect(formatPercentage(null)).toBe('—');
  });

  it('returns em-dash for undefined (no data)', () => {
    expect(formatPercentage(undefined)).toBe('—');
  });
});

// ---------------------------------------------------------------------------
// statusColor()
// ---------------------------------------------------------------------------

describe('statusColor()', () => {
  it('returns success color for "active"', () => {
    expect(statusColor('active')).toBe('text-status-success');
  });

  it('returns success color for "healthy"', () => {
    expect(statusColor('healthy')).toBe('text-status-success');
  });

  it('returns success color for "running"', () => {
    expect(statusColor('running')).toBe('text-status-success');
  });

  it('returns success color for "connected"', () => {
    expect(statusColor('connected')).toBe('text-status-success');
  });

  it('returns warning color for "warning"', () => {
    expect(statusColor('warning')).toBe('text-status-warning');
  });

  it('returns warning color for "degraded"', () => {
    expect(statusColor('degraded')).toBe('text-status-warning');
  });

  it('returns error color for "error"', () => {
    expect(statusColor('error')).toBe('text-status-error');
  });

  it('returns error color for "critical"', () => {
    expect(statusColor('critical')).toBe('text-status-error');
  });

  it('returns error color for "failed"', () => {
    expect(statusColor('failed')).toBe('text-status-error');
  });

  it('returns info color for "pending"', () => {
    expect(statusColor('pending')).toBe('text-status-info');
  });

  it('returns info color for "connecting"', () => {
    expect(statusColor('connecting')).toBe('text-status-info');
  });

  it('returns neutral color for "disconnected"', () => {
    expect(statusColor('disconnected')).toBe('text-status-neutral');
  });

  it('returns neutral color for "unknown"', () => {
    expect(statusColor('unknown')).toBe('text-status-neutral');
  });

  it('returns neutral color for unrecognized status', () => {
    expect(statusColor('something_random')).toBe('text-status-neutral');
  });

  it('is case insensitive', () => {
    expect(statusColor('Active')).toBe('text-status-success');
    expect(statusColor('ERROR')).toBe('text-status-error');
    expect(statusColor('Pending')).toBe('text-status-info');
  });
});

// ---------------------------------------------------------------------------
// statusBgColor()
// ---------------------------------------------------------------------------

describe('statusBgColor()', () => {
  it('returns success bg for "active"', () => {
    expect(statusBgColor('active')).toContain('bg-status-success');
  });

  it('returns error bg for "error"', () => {
    expect(statusBgColor('error')).toContain('bg-status-error');
  });

  it('returns default bg for unknown status', () => {
    expect(statusBgColor('xyz')).toContain('bg-status-neutral');
  });
});

// ---------------------------------------------------------------------------
// statusDotColor()
// ---------------------------------------------------------------------------

describe('statusDotColor()', () => {
  it('returns success dot for "active"', () => {
    expect(statusDotColor('active')).toBe('bg-status-success');
  });

  it('returns error dot for "failed"', () => {
    expect(statusDotColor('failed')).toBe('bg-status-error');
  });

  it('returns neutral dot for unknown', () => {
    expect(statusDotColor('xyz')).toBe('bg-status-neutral');
  });
});

// ---------------------------------------------------------------------------
// providerDisplayName()
// ---------------------------------------------------------------------------

describe('providerDisplayName()', () => {
  it('returns AWS for aws', () => {
    expect(providerDisplayName('aws')).toBe('AWS');
  });

  it('returns GCP for gcp', () => {
    expect(providerDisplayName('gcp')).toBe('GCP');
  });

  it('returns Azure for azure', () => {
    expect(providerDisplayName('azure')).toBe('Azure');
  });

  it('returns On-Premise for on-prem', () => {
    expect(providerDisplayName('on-prem')).toBe('On-Premise');
  });

  it('returns input for unknown provider', () => {
    expect(providerDisplayName('custom')).toBe('custom');
  });
});

// ---------------------------------------------------------------------------
// truncate()
// ---------------------------------------------------------------------------

describe('truncate()', () => {
  it('does not truncate short strings', () => {
    expect(truncate('hello', 10)).toBe('hello');
  });

  it('truncates long strings with ellipsis', () => {
    expect(truncate('hello world', 5)).toBe('hello...');
  });

  it('handles exact length', () => {
    expect(truncate('hello', 5)).toBe('hello');
  });
});

// ---------------------------------------------------------------------------
// stringToColor()
// ---------------------------------------------------------------------------

describe('stringToColor()', () => {
  it('returns an hsl string', () => {
    const result = stringToColor('test');
    expect(result).toMatch(/^hsl\(\d+, 60%, 50%\)$/);
  });

  it('is deterministic', () => {
    expect(stringToColor('foo')).toBe(stringToColor('foo'));
  });

  it('different strings produce different colors', () => {
    expect(stringToColor('alpha')).not.toBe(stringToColor('beta'));
  });
});

// ---------------------------------------------------------------------------
// gaugeColor() and gaugeTextColor()
// ---------------------------------------------------------------------------

describe('gaugeColor()', () => {
  it('returns success for low values', () => {
    expect(gaugeColor(50)).toBe('bg-status-success');
  });

  it('returns warning for 75+', () => {
    expect(gaugeColor(80)).toBe('bg-status-warning');
  });

  it('returns error for 90+', () => {
    expect(gaugeColor(95)).toBe('bg-status-error');
  });
});

describe('gaugeTextColor()', () => {
  it('returns success text for low values', () => {
    expect(gaugeTextColor(50)).toBe('text-status-success');
  });

  it('returns warning text for 75+', () => {
    expect(gaugeTextColor(80)).toBe('text-status-warning');
  });

  it('returns error text for 90+', () => {
    expect(gaugeTextColor(95)).toBe('text-status-error');
  });
});
