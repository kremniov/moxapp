import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { routesApi } from '@/lib/api';
import type { IncomingRouteRequest } from '@/types/api';

// List all routes
export function useRoutes() {
  return useQuery({
    queryKey: ['routes'],
    queryFn: routesApi.list,
    staleTime: 5000,
  });
}

// Get single route
export function useRoute(name: string) {
  return useQuery({
    queryKey: ['routes', name],
    queryFn: () => routesApi.get(name),
    enabled: !!name,
  });
}

// Create route
export function useCreateRoute() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: routesApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });
}

// Update route
export function useUpdateRoute() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ name, data }: { name: string; data: IncomingRouteRequest }) =>
      routesApi.update(name, data),
    onSuccess: (_, { name }) => {
      queryClient.invalidateQueries({ queryKey: ['routes'] });
      queryClient.invalidateQueries({ queryKey: ['routes', name] });
    },
  });
}

// Delete route
export function useDeleteRoute() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: routesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });
}

// Toggle route enabled state
export function useToggleRoute() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ name, enabled }: { name: string; enabled: boolean }) =>
      routesApi.setRouteEnabled(name, enabled),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['routes'] });
      queryClient.invalidateQueries({ queryKey: ['metrics'] });
    },
  });
}

// Get incoming control status
export function useIncomingControl() {
  return useQuery({
    queryKey: ['incoming', 'control'],
    queryFn: routesApi.getControl,
    staleTime: 5000,
  });
}

// Enable/disable all incoming routes
export function useToggleAllRoutes() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: routesApi.setEnabled,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['routes'] });
      queryClient.invalidateQueries({ queryKey: ['incoming'] });
      queryClient.invalidateQueries({ queryKey: ['metrics'] });
    },
  });
}

// Reload routes from config file
export function useReloadRoutes() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: routesApi.reload,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['routes'] });
    },
  });
}
