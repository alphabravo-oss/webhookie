import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';
import { format, formatDistanceToNow, parseISO } from 'date-fns';

/**
 * Merge Tailwind CSS classes with proper precedence
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/**
 * Format a date string to a human-readable format
 */
export function formatDate(dateStr: string, fmt: string = 'MMM d, yyyy HH:mm'): string {
  try {
    return format(parseISO(dateStr), fmt);
  } catch {
    return dateStr;
  }
}

/**
 * Format a date string to a relative time (e.g., "2 hours ago")
 */
export function formatRelativeTime(dateStr: string): string {
  try {
    return formatDistanceToNow(parseISO(dateStr), { addSuffix: true });
  } catch {
    return dateStr;
  }
}

/**
 * Format bytes to human-readable format (e.g., "1.5 GiB")
 */
export function formatBytes(bytes: number, decimals: number = 1): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(decimals))} ${sizes[i]}`;
}

/**
 * Format CPU millicores to human-readable format.
 *
 * Inputs can be raw floats from prometheus (e.g. 118.99999999999999 from
 * a rate() query that lost a hair of precision); the millicore branch
 * rounds to an integer so the UI never renders a 15-digit tail. The
 * cores branch caps at one decimal and strips trailing zeros so we get
 * "2 cores" not "2.0 cores".
 */
export function formatCPU(millicores: number): string {
  if (millicores == null || isNaN(millicores)) return '—';
  if (millicores >= 1000) {
    return `${parseFloat((millicores / 1000).toFixed(1))} cores`;
  }
  return `${Math.round(millicores)}m`;
}

/**
 * Format a percentage value. Returns "—" for null/undefined/NaN inputs so the
 * caller can distinguish "no data" from a real 0%. Trailing zeros after the
 * decimal point are stripped ("50%" not "50.0%").
 */
export function formatPercentage(value: number | undefined | null, decimals: number = 1): string {
  if (value == null || isNaN(value)) return '—';
  return `${parseFloat(value.toFixed(decimals))}%`;
}

/**
 * Get status color class based on status string
 */
export function statusColor(status: string): string {
  const normalized = status.toLowerCase().replace(/[\s_-]/g, '');
  const colorMap: Record<string, string> = {
    active: 'text-status-success',
    healthy: 'text-status-success',
    running: 'text-status-success',
    ready: 'text-status-success',
    synced: 'text-status-success',
    insync: 'text-status-success',
    succeeded: 'text-status-success',
    connected: 'text-status-success',
    success: 'text-status-success',
    allowed: 'text-status-success',
    permitted: 'text-status-success',
    enabled: 'text-status-success',
    compliant: 'text-status-success',

    warning: 'text-status-warning',
    degraded: 'text-status-warning',
    outofsync: 'text-status-warning',
    drifted: 'text-status-warning',
    stale: 'text-status-warning',
    readonly: 'text-status-warning',
    migrationrequired: 'text-status-warning',

    error: 'text-status-error',
    critical: 'text-status-error',
    failed: 'text-status-error',
    unhealthy: 'text-status-error',
    notready: 'text-status-error',
    denied: 'text-status-error',
    forbidden: 'text-status-error',
    blocked: 'text-status-error',
    missing: 'text-status-error',
    noncompliant: 'text-status-error',

    pending: 'text-status-info',
    connecting: 'text-status-info',
    provisioning: 'text-status-info',
    progressing: 'text-status-info',
    info: 'text-status-info',

    disconnected: 'text-status-neutral',
    unknown: 'text-status-neutral',
    suspended: 'text-status-neutral',
    disabled: 'text-status-neutral',
    unmanaged: 'text-status-neutral',
  };

  return colorMap[normalized] || 'text-status-neutral';
}

/**
 * Get status background color class
 */
export function statusBgColor(status: string): string {
  const normalized = status.toLowerCase().replace(/[\s_-]/g, '');
  const colorMap: Record<string, string> = {
    active: 'bg-status-success/10 text-status-success',
    healthy: 'bg-status-success/10 text-status-success',
    running: 'bg-status-success/10 text-status-success',
    ready: 'bg-status-success/10 text-status-success',
    synced: 'bg-status-success/10 text-status-success',
    insync: 'bg-status-success/10 text-status-success',
    succeeded: 'bg-status-success/10 text-status-success',
    connected: 'bg-status-success/10 text-status-success',
    success: 'bg-status-success/10 text-status-success',
    allowed: 'bg-status-success/10 text-status-success',
    permitted: 'bg-status-success/10 text-status-success',
    enabled: 'bg-status-success/10 text-status-success',
    compliant: 'bg-status-success/10 text-status-success',

    warning: 'bg-status-warning/10 text-status-warning',
    degraded: 'bg-status-warning/10 text-status-warning',
    outofsync: 'bg-status-warning/10 text-status-warning',
    drifted: 'bg-status-warning/10 text-status-warning',
    stale: 'bg-status-warning/10 text-status-warning',
    readonly: 'bg-status-warning/10 text-status-warning',
    migrationrequired: 'bg-status-warning/10 text-status-warning',
    decommissioning: 'bg-status-warning/10 text-status-warning',

    error: 'bg-status-error/10 text-status-error',
    critical: 'bg-status-error/10 text-status-error',
    failed: 'bg-status-error/10 text-status-error',
    unhealthy: 'bg-status-error/10 text-status-error',
    notready: 'bg-status-error/10 text-status-error',
    denied: 'bg-status-error/10 text-status-error',
    forbidden: 'bg-status-error/10 text-status-error',
    blocked: 'bg-status-error/10 text-status-error',
    missing: 'bg-status-error/10 text-status-error',
    noncompliant: 'bg-status-error/10 text-status-error',

    pending: 'bg-status-info/10 text-status-info',
    connecting: 'bg-status-info/10 text-status-info',
    provisioning: 'bg-status-info/10 text-status-info',
    progressing: 'bg-status-info/10 text-status-info',
    info: 'bg-status-info/10 text-status-info',

    disconnected: 'bg-status-neutral/10 text-status-neutral',
    unknown: 'bg-status-neutral/10 text-status-neutral',
    suspended: 'bg-status-neutral/10 text-status-neutral',
    disabled: 'bg-status-neutral/10 text-status-neutral',
    unmanaged: 'bg-status-neutral/10 text-status-neutral',
  };

  return colorMap[normalized] || 'bg-status-neutral/10 text-status-neutral';
}

/**
 * Get a dot indicator color class for status
 */
export function statusDotColor(status: string): string {
  const normalized = status.toLowerCase().replace(/[\s_-]/g, '');
  const colorMap: Record<string, string> = {
    active: 'bg-status-success',
    healthy: 'bg-status-success',
    running: 'bg-status-success',
    synced: 'bg-status-success',
    insync: 'bg-status-success',
    connected: 'bg-status-success',
    succeeded: 'bg-status-success',
    success: 'bg-status-success',
    allowed: 'bg-status-success',
    permitted: 'bg-status-success',
    enabled: 'bg-status-success',
    compliant: 'bg-status-success',

    warning: 'bg-status-warning',
    degraded: 'bg-status-warning',
    outofsync: 'bg-status-warning',
    drifted: 'bg-status-warning',
    stale: 'bg-status-warning',
    readonly: 'bg-status-warning',
    migrationrequired: 'bg-status-warning',
    decommissioning: 'bg-status-warning',

    error: 'bg-status-error',
    critical: 'bg-status-error',
    failed: 'bg-status-error',
    unhealthy: 'bg-status-error',
    notready: 'bg-status-error',
    denied: 'bg-status-error',
    forbidden: 'bg-status-error',
    blocked: 'bg-status-error',
    missing: 'bg-status-error',
    noncompliant: 'bg-status-error',

    pending: 'bg-status-info',
    connecting: 'bg-status-info',
    provisioning: 'bg-status-info',
    progressing: 'bg-status-info',
    info: 'bg-status-info',

    disconnected: 'bg-status-neutral',
    unknown: 'bg-status-neutral',
    suspended: 'bg-status-neutral',
    disabled: 'bg-status-neutral',
    unmanaged: 'bg-status-neutral',
  };

  return colorMap[normalized] || 'bg-status-neutral';
}

/**
 * Get provider display name
 */
export function providerDisplayName(provider: string): string {
  const names: Record<string, string> = {
    aws: 'AWS',
    gcp: 'GCP',
    azure: 'Azure',
    'on-prem': 'On-Premise',
    digitalocean: 'DigitalOcean',
    other: 'Other',
  };
  return names[provider] || provider;
}

/**
 * Convert a cluster distribution slug to a display name.
 */
export function distributionDisplayName(distribution: string): string {
  const names: Record<string, string> = {
    k3s: 'K3s',
    rke2: 'RKE2',
    eks: 'Amazon EKS',
    aks: 'Azure AKS',
    gke: 'Google GKE',
    openshift: 'OpenShift',
    k8s: 'Kubernetes',
  };
  return names[distribution] || distribution || 'Unknown';
}

/**
 * Truncate text with ellipsis
 */
export function truncate(str: string, maxLength: number): string {
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength) + '...';
}

/**
 * Generate a deterministic color from a string (for avatars, tags, etc.)
 */
export function stringToColor(str: string): string {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = Math.abs(hash % 360);
  return `hsl(${hue}, 60%, 50%)`;
}

/**
 * Copy text to clipboard
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

/**
 * Determine gauge color based on percentage thresholds
 */
export function gaugeColor(percentage: number): string {
  if (percentage >= 90) return 'bg-status-error';
  if (percentage >= 75) return 'bg-status-warning';
  return 'bg-status-success';
}

/**
 * Determine gauge text color based on percentage thresholds
 */
export function gaugeTextColor(percentage: number): string {
  if (percentage >= 90) return 'text-status-error';
  if (percentage >= 75) return 'text-status-warning';
  return 'text-status-success';
}

/**
 * Format a Kubernetes version for display.
 *
 * The agent reports the version straight from the Kubernetes API, which already
 * carries its own leading "v" ("v1.30.4+k3s1"). Call sites that hardcoded a `v`
 * prefix rendered "vv1.30.4+k3s1". Only add the v when it isn't already there.
 */
export function formatK8sVersion(version: string | null | undefined): string {
  const v = (version ?? '').trim();
  if (!v) return '—';
  return /^v/i.test(v) ? v : `v${v}`;
}
