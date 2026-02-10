import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { RefreshCw, Settings } from 'lucide-react';
import {
  useSettings,
  useUpdateMultiplier,
  useUpdateConcurrency,
  useUpdateLogRequests,
  useResetMetrics,
} from '@/hooks/use-metrics';
import { useState, useCallback } from 'react';
import { useToast } from '@/hooks/use-toast';

export function SettingsPanel() {
  const { data: settings, isLoading } = useSettings();
  const updateMultiplier = useUpdateMultiplier();
  const updateConcurrency = useUpdateConcurrency();
  const updateLogRequests = useUpdateLogRequests();
  const resetMetrics = useResetMetrics();
  const { toast } = useToast();

  // Local state for slider dragging (to avoid API calls during drag)
  const [pendingMultiplier, setPendingMultiplier] = useState<number | null>(null);
  const [pendingConcurrency, setPendingConcurrency] = useState<number | null>(null);

  // Use pending values while dragging/editing, otherwise use server values
  const displayMultiplier = pendingMultiplier ?? settings?.global_multiplier ?? 1;
  const displayConcurrency = pendingConcurrency ?? settings?.concurrent_requests ?? 20;
  const displayLogRequests = settings?.log_all_requests ?? false;

  const handleMultiplierChange = useCallback((value: number[]) => {
    setPendingMultiplier(value[0]);
  }, []);

  const handleMultiplierCommit = useCallback(() => {
    if (pendingMultiplier !== null) {
      updateMultiplier.mutate(pendingMultiplier, {
        onSuccess: () => {
          toast({ title: 'Multiplier updated', description: `Set to ${pendingMultiplier}x` });
          setPendingMultiplier(null);
        },
        onError: (err) => {
          toast({
            title: 'Failed to update multiplier',
            description: err.message,
            variant: 'destructive',
          });
          setPendingMultiplier(null);
        },
      });
    }
  }, [pendingMultiplier, updateMultiplier, toast]);

  const handleConcurrencyChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const value = parseInt(e.target.value, 10);
    if (!isNaN(value) && value > 0) {
      setPendingConcurrency(value);
    }
  }, []);

  const handleConcurrencyBlur = useCallback(() => {
    if (pendingConcurrency !== null && pendingConcurrency !== settings?.concurrent_requests) {
      updateConcurrency.mutate(pendingConcurrency, {
        onSuccess: () => {
          toast({
            title: 'Concurrency updated',
            description: `Set to ${pendingConcurrency} concurrent requests`,
          });
          setPendingConcurrency(null);
        },
        onError: (err) => {
          toast({
            title: 'Failed to update concurrency',
            description: err.message,
            variant: 'destructive',
          });
          setPendingConcurrency(null);
        },
      });
    } else {
      setPendingConcurrency(null);
    }
  }, [pendingConcurrency, settings?.concurrent_requests, updateConcurrency, toast]);

  const handleLogRequestsChange = useCallback((checked: boolean) => {
    updateLogRequests.mutate(checked, {
      onSuccess: () => {
        toast({
          title: 'Logging updated',
          description: checked ? 'Request logging enabled' : 'Request logging disabled',
        });
      },
      onError: (err) => {
        toast({
          title: 'Failed to update logging',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  }, [updateLogRequests, toast]);

  const handleResetMetrics = useCallback(() => {
    resetMetrics.mutate(undefined, {
      onSuccess: () => {
        toast({ title: 'Metrics reset', description: 'All metrics have been cleared' });
      },
      onError: (err) => {
        toast({
          title: 'Failed to reset metrics',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  }, [resetMetrics, toast]);

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Settings className="h-3.5 w-3.5" />
            Settings
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="animate-pulse space-y-4">
            <div className="h-8 bg-muted rounded" />
            <div className="h-8 bg-muted rounded" />
            <div className="h-8 bg-muted rounded" />
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Settings className="h-3.5 w-3.5" />
          Settings
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Multiplier */}
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <Label>Load Multiplier</Label>
            <span className="font-mono text-sm text-primary">{displayMultiplier}x</span>
          </div>
          <Slider
            value={[displayMultiplier]}
            onValueChange={handleMultiplierChange}
            onValueCommit={handleMultiplierCommit}
            min={0.1}
            max={10}
            step={0.1}
            className="w-full"
          />
          <p className="text-xs text-muted-foreground">
            Scales all endpoint frequencies
          </p>
        </div>

        {/* Concurrency */}
        <div className="space-y-2">
          <Label>Concurrent Requests</Label>
          <Input
            type="number"
            value={displayConcurrency}
            onChange={handleConcurrencyChange}
            onBlur={handleConcurrencyBlur}
            min={1}
            max={1000}
            className="w-full"
          />
          <p className="text-xs text-muted-foreground">
            Max parallel outgoing requests
          </p>
        </div>

        {/* Log Requests */}
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <Label>Log All Requests</Label>
            <p className="text-xs text-muted-foreground">
              Log individual request details
            </p>
          </div>
          <Switch checked={displayLogRequests} onCheckedChange={handleLogRequestsChange} />
        </div>

        {/* Reset Metrics */}
        <div className="pt-2 border-t">
          <Button
            variant="outline"
            size="sm"
            className="w-full"
            onClick={handleResetMetrics}
            disabled={resetMetrics.isPending}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${resetMetrics.isPending ? 'animate-spin' : ''}`} />
            Reset All Metrics
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
