import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { metricsApi, controlApi, settingsApi } from '@/lib/api';

// Live metrics - polled every 1 second
export function useLiveMetrics() {
  return useQuery({
    queryKey: ['metrics', 'overview'],
    queryFn: metricsApi.getOverview,
    refetchInterval: 1000,
    staleTime: 500,
  });
}

// Full metrics overview - polled every 2 seconds
export function useMetricsOverview() {
  return useQuery({
    queryKey: ['metrics', 'overview'],
    queryFn: metricsApi.getOverview,
    refetchInterval: 2000,
    staleTime: 1000,
  });
}

// Outgoing metrics only
export function useOutgoingMetrics() {
  return useQuery({
    queryKey: ['metrics', 'outgoing'],
    queryFn: metricsApi.getOutgoing,
    refetchInterval: 2000,
    staleTime: 1000,
  });
}

// Incoming metrics only
export function useIncomingMetrics() {
  return useQuery({
    queryKey: ['metrics', 'incoming'],
    queryFn: metricsApi.getIncoming,
    refetchInterval: 2000,
    staleTime: 1000,
  });
}

// Reset mutations
export function useResetMetrics() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: metricsApi.resetAll,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['metrics'] });
    },
  });
}

// Scheduler control
export function useSchedulerStatus() {
  return useQuery({
    queryKey: ['control', 'status'],
    queryFn: controlApi.getStatus,
    refetchInterval: 2000,
    staleTime: 1000,
  });
}

export function useSchedulerControl() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: controlApi.control,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['control', 'status'] });
      queryClient.invalidateQueries({ queryKey: ['metrics'] });
    },
  });
}

// Settings
export function useSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: settingsApi.getAll,
    staleTime: 30000,
  });
}

export function useUpdateMultiplier() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: settingsApi.setMultiplier,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}

export function useUpdateConcurrency() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: settingsApi.setConcurrency,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}

export function useUpdateLogRequests() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: settingsApi.setLogRequests,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] });
    },
  });
}
