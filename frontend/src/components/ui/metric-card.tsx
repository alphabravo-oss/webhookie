'use client';

import { cn, gaugeColor, gaugeTextColor } from '@/lib/utils';
import { ArrowUpRight, ArrowDownRight, Minus } from 'lucide-react';

interface MetricCardProps {
  title: string;
  value: string | number;
  unit?: string;
  subtitle?: string;
  trend?: 'up' | 'down' | 'flat';
  trendValue?: string;
  percentage?: number;
  thresholdWarning?: number;
  thresholdCritical?: number;
  icon?: React.ReactNode;
  sparkline?: number[];
  className?: string;
}

export function MetricCard({
  title,
  value,
  unit,
  subtitle,
  trend,
  trendValue,
  percentage,
  icon,
  sparkline,
  className,
}: MetricCardProps) {
  return (
    <div
      className={cn(
        'rounded-lg border border-border bg-card p-5 transition-colors hover:bg-card/80',
        className
      )}
    >
      <div className="flex items-start justify-between">
        <div className="space-y-1">
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          <div className="flex items-baseline gap-1.5">
            <span
              className={cn(
                'text-2xl font-semibold tracking-tight',
                percentage !== undefined ? gaugeTextColor(percentage) : 'text-foreground'
              )}
            >
              {value}
            </span>
            {unit && <span className="text-sm text-muted-foreground">{unit}</span>}
          </div>
          {subtitle && (
            <p className="text-xs text-muted-foreground">{subtitle}</p>
          )}
        </div>

        <div className="flex flex-col items-end gap-2">
          {icon && (
            <div className="rounded-md bg-muted p-2 text-muted-foreground">
              {icon}
            </div>
          )}

          {trend && trendValue && (
            <div
              className={cn(
                'flex items-center gap-0.5 text-xs font-medium',
                trend === 'up' && 'text-status-error',
                trend === 'down' && 'text-status-success',
                trend === 'flat' && 'text-muted-foreground'
              )}
            >
              {trend === 'up' && <ArrowUpRight className="h-3 w-3" />}
              {trend === 'down' && <ArrowDownRight className="h-3 w-3" />}
              {trend === 'flat' && <Minus className="h-3 w-3" />}
              {trendValue}
            </div>
          )}
        </div>
      </div>

      {/* Gauge bar for percentage values */}
      {percentage !== undefined && (
        <div className="mt-3">
          <div className="gauge-bar">
            <div
              className={cn('gauge-bar-fill', gaugeColor(percentage))}
              style={{ width: `${Math.min(percentage, 100)}%` }}
            />
          </div>
        </div>
      )}

      {/* Mini sparkline */}
      {sparkline && sparkline.length > 0 && (
        <div className="mt-3 flex items-end gap-px h-8">
          {sparkline.map((v, i) => {
            const max = Math.max(...sparkline);
            const height = max > 0 ? (v / max) * 100 : 0;
            return (
              <div
                key={i}
                className="flex-1 rounded-t-sm bg-primary/20 transition-all"
                style={{ height: `${Math.max(height, 4)}%` }}
              />
            );
          })}
        </div>
      )}
    </div>
  );
}
