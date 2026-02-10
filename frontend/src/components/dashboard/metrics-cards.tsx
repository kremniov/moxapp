import { Card, CardContent, CardTitle, CardHeader } from '@/components/ui/card';
import { cn, formatNumber, formatDuration, formatPercent } from '@/lib/utils';
import type { MetricsResponse } from '@/types/api';
import { Activity, CheckCircle2, XCircle, Clock, TrendingUp } from 'lucide-react';

interface MetricsCardsProps {
  metrics: MetricsResponse;
}

export function MetricsCards({ metrics }: MetricsCardsProps) {
  const totalSuccesses = metrics.outgoing.total_requests - metrics.outgoing.total_failures;
  
  const cards = [
    {
      title: 'Total Requests',
      value: formatNumber(metrics.outgoing.total_requests),
      icon: Activity,
      description: 'Outgoing requests made',
    },
    {
      title: 'Requests/Sec',
      value: formatNumber(metrics.outgoing.requests_per_sec, 1),
      icon: TrendingUp,
      description: 'Current throughput',
      highlight: true,
    },
    {
      title: 'Success Rate',
      value: formatPercent(metrics.outgoing.success_rate),
      icon: CheckCircle2,
      description: `${formatNumber(totalSuccesses)} successful`,
      color:
        metrics.outgoing.success_rate >= 99
          ? 'text-success'
          : metrics.outgoing.success_rate >= 95
          ? 'text-warning'
          : 'text-error',
    },
    {
      title: 'Failures',
      value: formatNumber(metrics.outgoing.total_failures),
      icon: XCircle,
      description: 'Failed requests',
      color: metrics.outgoing.total_failures > 0 ? 'text-error' : 'text-muted-foreground',
    },
    {
      title: 'Uptime',
      value: formatDuration(metrics.uptime_seconds),
      icon: Clock,
      description: 'Since scheduler start',
    },
    {
      title: 'Active Endpoints',
      value: `${metrics.outgoing.endpoint_count}`,
      icon: Activity,
      description: 'Enabled endpoints',
    },
  ];

  return (
    <div className="grid gap-4 grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
      {cards.map((card) => (
        <Card
          key={card.title}
          className={cn(
            'relative overflow-hidden',
            card.highlight && 'border-primary/50'
          )}
        >
          <CardHeader className="pb-2">
            <CardTitle className="flex items-center gap-2">
              <card.icon className="h-3.5 w-3.5" />
              {card.title}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div
              className={cn(
                'text-2xl font-mono font-bold tabular-nums',
                card.color || 'text-foreground'
              )}
            >
              {card.value}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              {card.description}
            </p>
          </CardContent>
          {card.highlight && (
            <div className="absolute inset-0 bg-gradient-to-br from-primary/5 to-transparent pointer-events-none" />
          )}
        </Card>
      ))}
    </div>
  );
}
