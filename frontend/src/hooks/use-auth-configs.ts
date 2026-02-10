import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { authApi } from '@/lib/api';
import type { AuthConfigRequest } from '@/types/api';

// List all auth configs
export function useAuthConfigs() {
  return useQuery({
    queryKey: ['auth-configs'],
    queryFn: authApi.list,
    staleTime: 10000,
  });
}

// Get single auth config
export function useAuthConfig(name: string) {
  return useQuery({
    queryKey: ['auth-configs', name],
    queryFn: () => authApi.get(name),
    enabled: !!name,
  });
}

// Create auth config
export function useCreateAuthConfig() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: authApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth-configs'] });
    },
  });
}

// Update auth config
export function useUpdateAuthConfig() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ name, data }: { name: string; data: AuthConfigRequest }) =>
      authApi.update(name, data),
    onSuccess: (_, { name }) => {
      queryClient.invalidateQueries({ queryKey: ['auth-configs'] });
      queryClient.invalidateQueries({ queryKey: ['auth-configs', name] });
    },
  });
}

// Delete auth config
export function useDeleteAuthConfig() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: authApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth-configs'] });
    },
  });
}

// Get token status
export function useTokenStatus(name: string) {
  return useQuery({
    queryKey: ['auth-configs', name, 'status'],
    queryFn: () => authApi.getTokenStatus(name),
    enabled: !!name,
    refetchInterval: 30000, // Refresh every 30 seconds
  });
}

// Refresh token
export function useRefreshToken() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: authApi.refreshToken,
    onSuccess: (_, name) => {
      queryClient.invalidateQueries({ queryKey: ['auth-configs', name, 'status'] });
    },
  });
}

// Set token manually
export function useSetToken() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ name, token, expiresIn }: { name: string; token: string; expiresIn?: number }) =>
      authApi.setToken(name, token, expiresIn),
    onSuccess: (_, { name }) => {
      queryClient.invalidateQueries({ queryKey: ['auth-configs', name, 'status'] });
    },
  });
}
