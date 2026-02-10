import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatNumber(num: number | undefined | null, decimals = 0): string {
  if (num == null || isNaN(num)) {
    return '0';
  }
  if (num >= 1_000_000) {
    return (num / 1_000_000).toFixed(1) + 'M';
  }
  if (num >= 1_000) {
    return (num / 1_000).toFixed(1) + 'K';
  }
  return num.toFixed(decimals);
}

export function formatDuration(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);
  
  if (hours > 0) {
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  }
  return `${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
}

export function formatMs(ms: number): string {
  if (ms < 1) return '<1ms';
  if (ms >= 1000) return (ms / 1000).toFixed(2) + 's';
  return ms.toFixed(0) + 'ms';
}

export function formatPercent(value: number | undefined | null): string {
  if (value == null || isNaN(value)) {
    return '0.0%';
  }
  return value.toFixed(1) + '%';
}

export function getSuccessRateColor(rate: number): string {
  if (rate >= 99) return 'text-success';
  if (rate >= 95) return 'text-warning';
  return 'text-error';
}

export function getMethodColor(method: string): string {
  const colors: Record<string, string> = {
    GET: 'text-chart-1',
    POST: 'text-chart-2',
    PUT: 'text-chart-3',
    DELETE: 'text-chart-5',
    PATCH: 'text-chart-4',
    HEAD: 'text-muted-foreground',
    OPTIONS: 'text-muted-foreground',
  };
  return colors[method.toUpperCase()] || 'text-foreground';
}

export function truncateUrl(url: string, maxLength = 50): string {
  if (url.length <= maxLength) return url;
  return url.substring(0, maxLength - 3) + '...';
}

export function timeAgo(date: string | Date): string {
  const now = new Date();
  const then = new Date(date);
  const seconds = Math.floor((now.getTime() - then.getTime()) / 1000);
  
  if (seconds < 5) return 'just now';
  if (seconds < 60) return `${seconds}s ago`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86400)}d ago`;
}
