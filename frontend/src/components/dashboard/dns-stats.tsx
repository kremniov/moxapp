import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { cn, formatNumber, formatMs } from '@/lib/utils';
import type { DomainSnapshot } from '@/types/api';
import { Globe } from 'lucide-react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ScrollArea } from '@/components/ui/scroll-area';

interface DnsStatsProps {
  stats: Record<string, DomainSnapshot>;
}

export function DnsStats({ stats }: DnsStatsProps) {
  const domains = Object.entries(stats || {});

  if (domains.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Globe className="h-3.5 w-3.5" />
            DNS Statistics by Domain
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground text-center py-8">
            No DNS statistics yet
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Globe className="h-3.5 w-3.5" />
          DNS Statistics by Domain
          <Badge variant="secondary" className="ml-auto">
            {domains.length} domains
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <ScrollArea className="h-[250px]">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Domain</TableHead>
                <TableHead className="text-right">Lookups</TableHead>
                <TableHead className="text-right">Success</TableHead>
                <TableHead className="text-right">Avg</TableHead>
                <TableHead className="text-right">P95</TableHead>
                <TableHead className="text-right">Max</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {domains.map(([domain, snapshot]) => {
                const successRate =
                  snapshot.total_lookups > 0
                    ? (snapshot.successful_lookups / snapshot.total_lookups) * 100
                    : 0;

                return (
                  <TableRow key={domain}>
                    <TableCell className="font-medium">
                      <span className="truncate block max-w-[200px]">{domain}</span>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {formatNumber(snapshot.total_lookups)}
                    </TableCell>
                    <TableCell className="text-right">
                      <span
                        className={cn(
                          'tabular-nums',
                          successRate >= 99
                            ? 'text-success'
                            : successRate >= 95
                            ? 'text-warning'
                            : 'text-error'
                        )}
                      >
                        {successRate.toFixed(1)}%
                      </span>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {formatMs(snapshot.avg_resolution_ms)}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {formatMs(snapshot.p95_resolution_ms)}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {formatMs(snapshot.max_resolution_ms)}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
