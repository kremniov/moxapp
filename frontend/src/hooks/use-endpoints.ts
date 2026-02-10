import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { endpointsApi, controlApi } from '@/lib/api';
import type { OutgoingEndpointRequest } from '@/types/api';

// List all endpoints
export function useEndpoints() {
  return useQuery({
    queryKey: ['endpoints'],
    queryFn: endpointsApi.list,
    staleTime: 5000,
  });
}

// Get single endpoint
export function useEndpoint(name: string) {
  return useQuery({
    queryKey: ['endpoints', name],
    queryFn: () => endpointsApi.get(name),
    enabled: !!name,
  });
}

// Create endpoint
export function useCreateEndpoint() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: endpointsApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['endpoints'] });
      queryClient.invalidateQueries({ queryKey: ['control'] });
    },
  });
}

// Update endpoint
export function useUpdateEndpoint() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ name, data }: { name: string; data: OutgoingEndpointRequest }) =>
      endpointsApi.update(name, data),
    onSuccess: (_, { name }) => {
      queryClient.invalidateQueries({ queryKey: ['endpoints'] });
      queryClient.invalidateQueries({ queryKey: ['endpoints', name] });
    },
  });
}

// Delete endpoint
export function useDeleteEndpoint() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: endpointsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['endpoints'] });
      queryClient.invalidateQueries({ queryKey: ['control'] });
    },
  });
}

// Toggle endpoint enabled state
export function useToggleEndpoint() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ name, enabled }: { name: string; enabled: boolean }) =>
      controlApi.setEndpointEnabled(name, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['endpoints'] });
      queryClient.invalidateQueries({ queryKey: ['control'] });
      queryClient.invalidateQueries({ queryKey: ['metrics'] });
    },
  });
}

// Enable/disable all endpoints
export function useToggleAllEndpoints() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: controlApi.setAllEndpointsEnabled,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['endpoints'] });
      queryClient.invalidateQueries({ queryKey: ['control'] });
      queryClient.invalidateQueries({ queryKey: ['metrics'] });
    },
  });
}

// Bulk delete endpoints
export function useBulkDeleteEndpoints() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: endpointsApi.bulkDelete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['endpoints'] });
      queryClient.invalidateQueries({ queryKey: ['control'] });
    },
  });
}
