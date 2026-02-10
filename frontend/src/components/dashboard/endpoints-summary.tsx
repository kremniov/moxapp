import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { cn, formatNumber, formatMs, formatPercent, getMethodColor } from '@/lib/utils';
import type { MetricsSnapshot, EndpointSnapshot } from '@/types/api';
import { Send, CheckCircle2, XCircle, Clock } from 'lucide-react';
import { ScrollArea } from '@/components/ui/scroll-area';

interface EndpointsSummaryProps {
  metrics: MetricsSnapshot;
}

export function EndpointsSummary({ metrics }: EndpointsSummaryProps) {
  const endpoints = Object.entries(metrics.endpoints || {});

  if (endpoints.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Send className="h-3.5 w-3.5" />
            Outgoing Endpoints
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground text-center py-8">
            No endpoint metrics yet
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Send className="h-3.5 w-3.5" />
          Outgoing Endpoints
          <Badge variant="secondary" className="ml-auto">
            {endpoints.length}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <ScrollArea className="h-[400px]">
          <div className="divide-y">
            {endpoints.map(([name, snapshot]) => (
              <EndpointRow key={name} name={name} snapshot={snapshot} />
            ))}
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}

function EndpointRow({
  name,
  snapshot,
}: {
  name: string;
  snapshot: EndpointSnapshot;
}) {
  const successRate = snapshot.success_rate || 0;
  const method = snapshot.url_pattern?.match(/^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)/)?.[0] || 'GET';

  return (
    <div className="p-4 hover:bg-muted/50 transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-mono text-sm font-medium truncate">{name}</span>
            <span className={cn('font-mono text-xs', getMethodColor(method))}>
              {method}
            </span>
          </div>
          <p className="text-xs text-muted-foreground truncate mt-0.5">
            {snapshot.hostname || snapshot.url_pattern || 'N/A'}
          </p>
        </div>

        <div className="flex items-center gap-4 text-right">
          {/* Request counts */}
          <div className="text-xs space-y-1">
            <div className="flex items-center gap-1 justify-end">
              <CheckCircle2 className="h-3 w-3 text-success" />
              <span className="font-mono tabular-nums">{formatNumber(snapshot.successful)}</span>
            </div>
            <div className="flex items-center gap-1 justify-end">
              <XCircle className="h-3 w-3 text-error" />
              <span className="font-mono tabular-nums">{formatNumber(snapshot.failed)}</span>
            </div>
          </div>

          {/* Latency */}
          <div className="text-xs space-y-1 w-20">
            <div className="flex items-center gap-1 justify-end">
              <Clock className="h-3 w-3 text-muted-foreground" />
              <span className="font-mono tabular-nums">{formatMs(snapshot.avg_total_time_ms)}</span>
            </div>
            <div className="text-muted-foreground">
              <span className="font-mono tabular-nums">p95: {formatMs(snapshot.p95_total_time_ms)}</span>
            </div>
          </div>

          {/* Success Rate */}
          <div className="w-16 text-right">
            <span
              className={cn(
                'font-mono text-sm font-medium tabular-nums',
                successRate >= 99 ? 'text-success' : successRate >= 95 ? 'text-warning' : 'text-error'
              )}
            >
              {formatPercent(successRate)}
            </span>
          </div>
        </div>
      </div>

      {/* Progress bar */}
      <div className="mt-2 h-1 bg-muted rounded-full overflow-hidden">
        <div
          className={cn(
            'h-full transition-all',
            successRate >= 99 ? 'bg-success' : successRate >= 95 ? 'bg-warning' : 'bg-error'
          )}
          style={{ width: `${successRate}%` }}
        />
      </div>

      {/* Error breakdown if any */}
      {snapshot.failed > 0 && (
        <div className="mt-2 flex gap-2 text-xs">
          {snapshot.timeout_errors > 0 && (
            <Badge variant="error">Timeout: {snapshot.timeout_errors}</Badge>
          )}
          {snapshot.dns_errors > 0 && (
            <Badge variant="error">DNS: {snapshot.dns_errors}</Badge>
          )}
          {snapshot.connection_errors > 0 && (
            <Badge variant="error">Conn: {snapshot.connection_errors}</Badge>
          )}
          {snapshot.http_errors > 0 && (
            <Badge variant="error">HTTP: {snapshot.http_errors}</Badge>
          )}
        </div>
      )}
    </div>
  );
}
