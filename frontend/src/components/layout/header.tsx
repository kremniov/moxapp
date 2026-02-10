import { Pause, Play, StopCircle, Sun, Moon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { useTheme } from '@/hooks/use-theme';
import { useLiveMetrics, useSchedulerControl, useSchedulerStatus } from '@/hooks/use-metrics';
import { cn, formatNumber, formatPercent } from '@/lib/utils';
import type { SchedulerState } from '@/types/api';

const stateConfig: Record<
  SchedulerState,
  { label: string; color: string; bgColor: string }
> = {
  running: {
    label: 'RUNNING',
    color: 'text-running',
    bgColor: 'bg-running',
  },
  paused: {
    label: 'PAUSED',
    color: 'text-paused',
    bgColor: 'bg-paused',
  },
  stopped: {
    label: 'STOPPED',
    color: 'text-stopped',
    bgColor: 'bg-stopped',
  },
};

export function Header() {
  const { theme, toggleTheme } = useTheme();
  const { data: metrics, isLoading } = useLiveMetrics();
  const { data: controlStatus } = useSchedulerStatus();
  const controlMutation = useSchedulerControl();

  const state: SchedulerState = controlStatus
    ? controlStatus.paused
      ? 'paused'
      : controlStatus.scheduler_running && controlStatus.global_enabled
        ? 'running'
        : 'stopped'
    : 'stopped';
  const config = stateConfig[state];

  const handleControl = (action: 'pause' | 'resume' | 'emergency_stop') => {
    controlMutation.mutate(action);
  };

  return (
    <header className="flex h-14 items-center justify-between border-b bg-card px-4">
      {/* Status Section */}
      <div className="flex items-center gap-6">
        {/* State Indicator */}
        <div className="flex items-center gap-2">
          <div
            className={cn(
              'h-2.5 w-2.5 rounded-full',
              config.bgColor,
              state === 'running' && 'status-pulse'
            )}
          />
          <span className={cn('font-mono text-sm font-medium', config.color)}>
            {config.label}
          </span>
        </div>

        {/* Quick Stats */}
        {!isLoading && metrics && (
          <div className="flex items-center gap-4 text-sm font-mono">
            <div className="flex items-center gap-1.5">
              <span className="text-muted-foreground text-xs">RPS</span>
              <span className="text-foreground tabular-nums">
                {formatNumber(metrics.outgoing.requests_per_sec, 1)}
              </span>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="text-muted-foreground text-xs">SUCCESS</span>
              <span
                className={cn(
                  'tabular-nums',
                  metrics.outgoing.success_rate >= 99
                    ? 'text-success'
                    : metrics.outgoing.success_rate >= 95
                    ? 'text-warning'
                    : 'text-error'
                )}
              >
                {formatPercent(metrics.outgoing.success_rate)}
              </span>
            </div>
            <div className="flex items-center gap-1.5">
              <span className="text-muted-foreground text-xs">TOTAL</span>
              <span className="text-foreground tabular-nums">
                {formatNumber(metrics.outgoing.total_requests)}
              </span>
            </div>
          </div>
        )}
      </div>

      {/* Controls Section */}
      <div className="flex items-center gap-2">
        {/* Scheduler Controls */}
        <div className="flex items-center gap-1 mr-2">
          {state === 'running' ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => handleControl('pause')}
                  disabled={controlMutation.isPending}
                >
                  <Pause className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Pause scheduler</TooltipContent>
            </Tooltip>
          ) : state === 'paused' ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => handleControl('resume')}
                  disabled={controlMutation.isPending}
                >
                  <Play className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Resume scheduler</TooltipContent>
            </Tooltip>
          ) : (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => handleControl('resume')}
                  disabled={controlMutation.isPending}
                >
                  <Play className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Start scheduler</TooltipContent>
            </Tooltip>
          )}

          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="outline"
                size="icon"
                onClick={() => handleControl('emergency_stop')}
                disabled={controlMutation.isPending || state === 'stopped'}
                className="text-destructive hover:text-destructive"
              >
                <StopCircle className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Emergency stop</TooltipContent>
          </Tooltip>
        </div>

        {/* Theme Toggle */}
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon" onClick={toggleTheme}>
              {theme === 'dark' ? (
                <Sun className="h-4 w-4" />
              ) : (
                <Moon className="h-4 w-4" />
              )}
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            Switch to {theme === 'dark' ? 'light' : 'dark'} mode
          </TooltipContent>
        </Tooltip>
      </div>
    </header>
  );
}
