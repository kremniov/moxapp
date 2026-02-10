import { useLiveMetrics, useOutgoingMetrics, useIncomingMetrics } from '@/hooks/use-metrics';
import { MetricsCards } from '@/components/dashboard/metrics-cards';
import { SettingsPanel } from '@/components/dashboard/settings-panel';
import { EndpointsSummary } from '@/components/dashboard/endpoints-summary';
import { RoutesSummary } from '@/components/dashboard/routes-summary';
import { RpsChart, SuccessRateChart } from '@/components/dashboard/metrics-charts';
import { DnsStats } from '@/components/dashboard/dns-stats';
import { Skeleton } from '@/components/ui/skeleton';
import { Card, CardContent, CardHeader } from '@/components/ui/card';

export function DashboardPage() {
  const { data: liveMetrics, isLoading: liveLoading } = useLiveMetrics();
  const { data: outgoingMetrics, isLoading: outgoingLoading } = useOutgoingMetrics();
  const { data: incomingMetrics, isLoading: incomingLoading } = useIncomingMetrics();

  if (liveLoading || !liveMetrics) {
    return <DashboardSkeleton />;
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Dashboard</h1>
        <p className="text-sm text-muted-foreground">
          Real-time monitoring of load test metrics
        </p>
      </div>

      {/* Metrics Overview Cards */}
      <MetricsCards metrics={liveMetrics} />

      {/* Charts Row */}
      <div className="grid gap-4 lg:grid-cols-2">
        <RpsChart
          rps={liveMetrics.outgoing.requests_per_sec}
          successRate={liveMetrics.outgoing.success_rate}
        />
        <SuccessRateChart successRate={liveMetrics.outgoing.success_rate} />
      </div>

      {/* Main Content Grid */}
      <div className="grid gap-4 lg:grid-cols-3">
        {/* Left column - Endpoints and Routes */}
        <div className="lg:col-span-2 space-y-4">
          {outgoingMetrics && <EndpointsSummary metrics={outgoingMetrics} />}
          {incomingMetrics && <RoutesSummary metrics={incomingMetrics} />}
          {(outgoingLoading || incomingLoading) && (
            <Card>
              <CardHeader>
                <Skeleton className="h-4 w-40" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-[300px] w-full" />
              </CardContent>
            </Card>
          )}
        </div>

        {/* Right column - Settings and DNS Stats */}
        <div className="space-y-4">
          <SettingsPanel />
          {outgoingMetrics && <DnsStats stats={outgoingMetrics.dns_stats_by_domain} />}
        </div>
      </div>
    </div>
  );
}

function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      <div>
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-4 w-64 mt-2" />
      </div>

      {/* Metrics cards skeleton */}
      <div className="grid gap-4 grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
        {Array.from({ length: 6 }).map((_, i) => (
          <Card key={i}>
            <CardHeader className="pb-2">
              <Skeleton className="h-3 w-20" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-16" />
              <Skeleton className="h-3 w-24 mt-2" />
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Charts skeleton */}
      <div className="grid gap-4 lg:grid-cols-2">
        {Array.from({ length: 2 }).map((_, i) => (
          <Card key={i}>
            <CardHeader>
              <Skeleton className="h-4 w-32" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-[200px] w-full" />
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
