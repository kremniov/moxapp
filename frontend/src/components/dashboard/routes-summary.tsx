import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { formatNumber, formatMs } from '@/lib/utils';
import type { IncomingMetricsSnapshot, IncomingRouteSnapshot } from '@/types/api';
import { Inbox, Clock, Activity } from 'lucide-react';
import { ScrollArea } from '@/components/ui/scroll-area';

interface RoutesSummaryProps {
  metrics: IncomingMetricsSnapshot;
}

export function RoutesSummary({ metrics }: RoutesSummaryProps) {
  const routes = Object.entries(metrics.routes || {});

  if (routes.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Inbox className="h-3.5 w-3.5" />
            Incoming Routes
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground text-center py-8">
            No incoming route metrics yet
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Inbox className="h-3.5 w-3.5" />
          Incoming Routes
          <Badge variant="secondary" className="ml-auto">
            {routes.length}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <ScrollArea className="h-[300px]">
          <div className="divide-y">
            {routes.map(([name, snapshot]) => (
              <RouteRow key={name} snapshot={snapshot} />
            ))}
          </div>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}

function RouteRow({ snapshot }: { snapshot: IncomingRouteSnapshot }) {
  const statusEntries = Object.entries(snapshot.responses_by_status || {});

  return (
    <div className="p-4 hover:bg-muted/50 transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-mono text-sm font-medium">{snapshot.route_name}</span>
          </div>
          <p className="text-xs text-muted-foreground font-mono mt-0.5">
            {snapshot.route_path}
          </p>
        </div>

        <div className="flex items-center gap-4 text-right">
          {/* Request count */}
          <div className="text-xs space-y-1">
            <div className="flex items-center gap-1 justify-end">
              <Activity className="h-3 w-3 text-primary" />
              <span className="font-mono tabular-nums">
                {formatNumber(snapshot.total_requests)}
              </span>
            </div>
          </div>

          {/* Latency */}
          <div className="text-xs space-y-1 w-20">
            <div className="flex items-center gap-1 justify-end">
              <Clock className="h-3 w-3 text-muted-foreground" />
              <span className="font-mono tabular-nums">
                {formatMs(snapshot.avg_response_ms)}
              </span>
            </div>
            <div className="text-muted-foreground">
              <span className="font-mono tabular-nums">
                p95: {formatMs(snapshot.p95_response_ms)}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Status code breakdown */}
      {statusEntries.length > 0 && (
        <div className="mt-2 flex gap-2 flex-wrap">
          {statusEntries.map(([status, count]) => {
            const statusNum = parseInt(status, 10);
            const variant =
              statusNum >= 200 && statusNum < 300
                ? 'success'
                : statusNum >= 400 && statusNum < 500
                ? 'warning'
                : statusNum >= 500
                ? 'error'
                : 'secondary';
            return (
              <Badge key={status} variant={variant}>
                {status}: {formatNumber(count as number)}
              </Badge>
            );
          })}
        </div>
      )}
    </div>
  );
}
