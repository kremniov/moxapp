import { useState } from 'react';
import {
  useAuthConfigs,
  useCreateAuthConfig,
  useUpdateAuthConfig,
  useDeleteAuthConfig,
  useRefreshToken,
} from '@/hooks/use-auth-configs';
import { useToast } from '@/hooks/use-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Plus,
  Pencil,
  Trash2,
  KeyRound,
  RefreshCw,
} from 'lucide-react';
import { AuthForm } from '@/components/admin/auth-form';
import type { AuthConfig, AuthConfigRequest } from '@/types/api';

export function AdminAuthPage() {
  const { data: authConfigs, isLoading } = useAuthConfigs();
  const createAuthConfig = useCreateAuthConfig();
  const updateAuthConfig = useUpdateAuthConfig();
  const deleteAuthConfig = useDeleteAuthConfig();
  const refreshToken = useRefreshToken();
  const { toast } = useToast();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editingConfig, setEditingConfig] = useState<AuthConfig | null>(null);
  const [deleteConfirmConfig, setDeleteConfirmConfig] = useState<string | null>(null);

  const handleCreate = (data: AuthConfigRequest) => {
    createAuthConfig.mutate(data, {
      onSuccess: () => {
        setCreateDialogOpen(false);
        toast({ title: 'Auth config created', description: `${data.name} has been created` });
      },
      onError: (err) => {
        toast({
          title: 'Failed to create auth config',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  };

  const handleUpdate = (data: AuthConfigRequest) => {
    if (!editingConfig) return;
    updateAuthConfig.mutate(
      { name: editingConfig.name, data },
      {
        onSuccess: () => {
          setEditingConfig(null);
          toast({ title: 'Auth config updated', description: `${data.name} has been updated` });
        },
        onError: (err) => {
          toast({
            title: 'Failed to update auth config',
            description: err.message,
            variant: 'destructive',
          });
        },
      }
    );
  };

  const handleDelete = () => {
    if (!deleteConfirmConfig) return;
    deleteAuthConfig.mutate(deleteConfirmConfig, {
      onSuccess: () => {
        setDeleteConfirmConfig(null);
        toast({ title: 'Auth config deleted' });
      },
      onError: (err) => {
        toast({
          title: 'Failed to delete auth config',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  };

  const handleRefreshToken = (name: string) => {
    refreshToken.mutate(name, {
      onSuccess: () => {
        toast({ title: 'Token refreshed', description: `Token for ${name} has been refreshed` });
      },
      onError: (err) => {
        toast({
          title: 'Failed to refresh token',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  };

  const getAuthTypeBadgeVariant = (type: string) => {
    switch (type) {
      case 'bearer':
        return 'default';
      case 'api_key':
      case 'api_key_query':
        return 'secondary';
      case 'basic':
        return 'outline';
      case 'custom_header':
        return 'secondary';
      default:
        return 'outline';
    }
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Authentication Configs</h1>
          <p className="text-sm text-muted-foreground">
            Manage authentication configurations for outgoing endpoints
          </p>
        </div>
        <Button onClick={() => setCreateDialogOpen(true)}>
          <Plus className="h-4 w-4 mr-1" />
          Add Auth Config
        </Button>
      </div>

      {/* Auth Configs Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <KeyRound className="h-4 w-4" />
            Auth Configurations
            <Badge variant="secondary" className="ml-auto">
              {authConfigs?.length || 0} configs
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-4 space-y-4">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : !authConfigs || authConfigs.length === 0 ? (
            <div className="p-8 text-center">
              <KeyRound className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">No auth configurations</p>
              <Button
                variant="outline"
                className="mt-4"
                onClick={() => setCreateDialogOpen(true)}
              >
                <Plus className="h-4 w-4 mr-1" />
                Add your first auth config
              </Button>
            </div>
          ) : (
            <ScrollArea className="h-[500px]">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Description</TableHead>
                    <TableHead>Details</TableHead>
                    <TableHead className="w-32">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {authConfigs.map((config) => (
                    <TableRow key={config.name}>
                      <TableCell className="font-medium font-mono">
                        {config.name}
                      </TableCell>
                      <TableCell>
                        <Badge variant={getAuthTypeBadgeVariant(config.type)}>
                          {config.type}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-muted-foreground text-sm max-w-[200px] truncate">
                        {config.description || '-'}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground font-mono">
                        {config.env_var && <div>env: {config.env_var}</div>}
                        {config.header_name && <div>header: {config.header_name}</div>}
                        {config.query_param && <div>param: {config.query_param}</div>}
                        {config.token_endpoint && <div>token endpoint configured</div>}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          {config.type === 'bearer' && config.token_endpoint && (
                            <Button
                              variant="ghost"
                              size="icon"
                              onClick={() => handleRefreshToken(config.name)}
                              disabled={refreshToken.isPending}
                            >
                              <RefreshCw
                                className={`h-4 w-4 ${
                                  refreshToken.isPending ? 'animate-spin' : ''
                                }`}
                              />
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setEditingConfig(config)}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setDeleteConfirmConfig(config.name)}
                          >
                            <Trash2 className="h-4 w-4 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </ScrollArea>
          )}
        </CardContent>
      </Card>

      {/* Create Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Create Auth Config</DialogTitle>
            <DialogDescription>
              Add a new authentication configuration
            </DialogDescription>
          </DialogHeader>
          <AuthForm
            onSubmit={handleCreate}
            onCancel={() => setCreateDialogOpen(false)}
            isLoading={createAuthConfig.isPending}
          />
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editingConfig} onOpenChange={() => setEditingConfig(null)}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Auth Config</DialogTitle>
            <DialogDescription>Modify authentication configuration</DialogDescription>
          </DialogHeader>
          {editingConfig && (
            <AuthForm
              initialData={editingConfig}
              onSubmit={handleUpdate}
              onCancel={() => setEditingConfig(null)}
              isLoading={updateAuthConfig.isPending}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={!!deleteConfirmConfig}
        onOpenChange={() => setDeleteConfirmConfig(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Auth Config</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete "{deleteConfirmConfig}"? Endpoints using
              this auth config will no longer work properly.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirmConfig(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteAuthConfig.isPending}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
