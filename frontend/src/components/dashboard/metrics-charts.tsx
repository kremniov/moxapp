import { useSyncExternalStore, useEffect, useMemo, useRef } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
} from 'recharts';
import { TrendingUp } from 'lucide-react';

interface RpsChartProps {
  rps: number;
  successRate: number;
}

const MAX_DATA_POINTS = 60; // 1 minute of data at 1s intervals

// External store for RPS chart data
type ChartDataPoint = { time: string; rps: number; successRate: number };
let rpsChartData: ChartDataPoint[] = [];
let rpsChartListeners: Array<() => void> = [];
let rpsLastUpdate = 0;
let rpsLastValues = { rps: 0, successRate: 0 };

function subscribeRpsChart(callback: () => void) {
  rpsChartListeners.push(callback);
  return () => {
    rpsChartListeners = rpsChartListeners.filter(l => l !== callback);
  };
}

function getRpsChartSnapshot() {
  return rpsChartData;
}

function updateRpsChart(rps: number, successRate: number) {
  // Skip if values haven't changed
  if (rps === rpsLastValues.rps && successRate === rpsLastValues.successRate) {
    return;
  }
  rpsLastValues = { rps, successRate };
  
  const now = Date.now();
  if (now - rpsLastUpdate >= 900) {
    rpsLastUpdate = now;
    const timeStr = new Date().toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
    rpsChartData = [
      ...rpsChartData.slice(-MAX_DATA_POINTS + 1),
      { time: timeStr, rps, successRate },
    ];
    rpsChartListeners.forEach(l => l());
  }
}

export function RpsChart({ rps, successRate }: RpsChartProps) {
  const hasMounted = useRef(false);
  const displayData = useSyncExternalStore(subscribeRpsChart, getRpsChartSnapshot);
  
  // Update external store when props change (using effect to satisfy lint rules)
  useEffect(() => {
    updateRpsChart(rps, successRate);
  }, [rps, successRate]);

  useEffect(() => {
    hasMounted.current = true;
    return () => {
      hasMounted.current = false;
    };
  }, []);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TrendingUp className="h-3.5 w-3.5" />
          Requests per Second
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-[200px] min-h-[200px] min-w-0">
          {hasMounted.current && displayData.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%" minWidth={300} minHeight={200}>
              <AreaChart data={displayData}>
              <defs>
                <linearGradient id="rpsGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="oklch(70% 0.15 190)" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="oklch(70% 0.15 190)" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid
                strokeDasharray="3 3"
                stroke="oklch(25% 0 0)"
                vertical={false}
              />
              <XAxis
                dataKey="time"
                stroke="oklch(60% 0 0)"
                fontSize={10}
                tickLine={false}
                axisLine={false}
                interval="preserveStartEnd"
                minTickGap={50}
              />
              <YAxis
                stroke="oklch(60% 0 0)"
                fontSize={10}
                tickLine={false}
                axisLine={false}
                width={40}
                tickFormatter={(value) => value.toFixed(0)}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'oklch(16% 0 0)',
                  border: '1px solid oklch(25% 0 0)',
                  borderRadius: '6px',
                  fontFamily: 'JetBrains Mono, monospace',
                  fontSize: '12px',
                }}
                labelStyle={{ color: 'oklch(90% 0 0)' }}
                itemStyle={{ color: 'oklch(70% 0.15 190)' }}
              />
              <Area
                type="monotone"
                dataKey="rps"
                stroke="oklch(70% 0.15 190)"
                strokeWidth={2}
                fill="url(#rpsGradient)"
                dot={false}
                activeDot={{ r: 4, fill: 'oklch(70% 0.15 190)' }}
                isAnimationActive={false}
              />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-full w-full" />
          )}
        </div>
      </CardContent>
    </Card>
  );
}

interface SuccessRateChartProps {
  successRate: number;
}

// External store for success rate chart data
type SuccessDataPoint = { time: string; rate: number };
let successChartData: SuccessDataPoint[] = [];
let successChartListeners: Array<() => void> = [];
let successLastUpdate = 0;
let successLastValue = 0;

function subscribeSuccessChart(callback: () => void) {
  successChartListeners.push(callback);
  return () => {
    successChartListeners = successChartListeners.filter(l => l !== callback);
  };
}

function getSuccessChartSnapshot() {
  return successChartData;
}

function updateSuccessChart(successRate: number) {
  // Skip if value hasn't changed
  if (successRate === successLastValue) {
    return;
  }
  successLastValue = successRate;
  
  const now = Date.now();
  if (now - successLastUpdate >= 900) {
    successLastUpdate = now;
    const timeStr = new Date().toLocaleTimeString('en-US', {
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
    successChartData = [
      ...successChartData.slice(-MAX_DATA_POINTS + 1),
      { time: timeStr, rate: successRate },
    ];
    successChartListeners.forEach(l => l());
  }
}

export function SuccessRateChart({ successRate }: SuccessRateChartProps) {
  const hasMounted = useRef(false);
  const displayData = useSyncExternalStore(subscribeSuccessChart, getSuccessChartSnapshot);
  
  // Update external store when props change (using effect to satisfy lint rules)
  useEffect(() => {
    updateSuccessChart(successRate);
  }, [successRate]);

  useEffect(() => {
    hasMounted.current = true;
    return () => {
      hasMounted.current = false;
    };
  }, []);

  const currentColor = useMemo(() => {
    if (successRate >= 99) return 'oklch(65% 0.18 145)'; // success
    if (successRate >= 95) return 'oklch(75% 0.18 85)'; // warning
    return 'oklch(55% 0.22 25)'; // error
  }, [successRate]);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TrendingUp className="h-3.5 w-3.5" />
          Success Rate
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="h-[200px] min-h-[200px] min-w-0">
          {hasMounted.current && displayData.length > 0 ? (
            <ResponsiveContainer width="100%" height="100%" minWidth={300} minHeight={200}>
              <LineChart data={displayData}>
              <CartesianGrid
                strokeDasharray="3 3"
                stroke="oklch(25% 0 0)"
                vertical={false}
              />
              <XAxis
                dataKey="time"
                stroke="oklch(60% 0 0)"
                fontSize={10}
                tickLine={false}
                axisLine={false}
                interval="preserveStartEnd"
                minTickGap={50}
              />
              <YAxis
                stroke="oklch(60% 0 0)"
                fontSize={10}
                tickLine={false}
                axisLine={false}
                width={40}
                domain={[
                  (dataMin: number) => Math.max(0, Math.floor(dataMin - 5)),
                  100,
                ]}
                tickFormatter={(value) => `${value}%`}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'oklch(16% 0 0)',
                  border: '1px solid oklch(25% 0 0)',
                  borderRadius: '6px',
                  fontFamily: 'JetBrains Mono, monospace',
                  fontSize: '12px',
                }}
                labelStyle={{ color: 'oklch(90% 0 0)' }}
                formatter={(value) => value != null ? [`${Number(value).toFixed(2)}%`, 'Success Rate'] : ['N/A', 'Success Rate']}
              />
              <Line
                type="monotone"
                dataKey="rate"
                stroke={currentColor}
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, fill: currentColor }}
                isAnimationActive={false}
              />
              </LineChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-full w-full" />
          )}
        </div>
      </CardContent>
    </Card>
  );
}
