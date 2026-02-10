import { useState } from 'react';
import {
  useEndpoints,
  useCreateEndpoint,
  useUpdateEndpoint,
  useDeleteEndpoint,
  useToggleEndpoint,
  useToggleAllEndpoints,
} from '@/hooks/use-endpoints';
import { useToast } from '@/hooks/use-toast';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
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
  Send,
  Power,
  PowerOff,
} from 'lucide-react';
import { cn, getMethodColor, truncateUrl } from '@/lib/utils';
import { EndpointForm } from '@/components/admin/endpoint-form';
import type { OutgoingEndpoint, OutgoingEndpointRequest } from '@/types/api';

export function AdminEndpointsPage() {
  const { data: endpoints, isLoading } = useEndpoints();
  const createEndpoint = useCreateEndpoint();
  const updateEndpoint = useUpdateEndpoint();
  const deleteEndpoint = useDeleteEndpoint();
  const toggleEndpoint = useToggleEndpoint();
  const toggleAllEndpoints = useToggleAllEndpoints();
  const { toast } = useToast();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editingEndpoint, setEditingEndpoint] = useState<OutgoingEndpoint | null>(null);
  const [deleteConfirmEndpoint, setDeleteConfirmEndpoint] = useState<string | null>(null);

  const handleCreate = (data: OutgoingEndpointRequest) => {
    createEndpoint.mutate(data, {
      onSuccess: () => {
        setCreateDialogOpen(false);
        toast({ title: 'Endpoint created', description: `${data.name} has been created` });
      },
      onError: (err) => {
        toast({
          title: 'Failed to create endpoint',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  };

  const handleUpdate = (data: OutgoingEndpointRequest) => {
    if (!editingEndpoint) return;
    updateEndpoint.mutate(
      { name: editingEndpoint.name, data },
      {
        onSuccess: () => {
          setEditingEndpoint(null);
          toast({ title: 'Endpoint updated', description: `${data.name} has been updated` });
        },
        onError: (err) => {
          toast({
            title: 'Failed to update endpoint',
            description: err.message,
            variant: 'destructive',
          });
        },
      }
    );
  };

  const handleDelete = () => {
    if (!deleteConfirmEndpoint) return;
    deleteEndpoint.mutate(deleteConfirmEndpoint, {
      onSuccess: () => {
        setDeleteConfirmEndpoint(null);
        toast({ title: 'Endpoint deleted' });
      },
      onError: (err) => {
        toast({
          title: 'Failed to delete endpoint',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  };

  const handleToggle = (name: string, enabled: boolean) => {
    toggleEndpoint.mutate(
      { name, enabled },
      {
        onError: (err) => {
          toast({
            title: 'Failed to toggle endpoint',
            description: err.message,
            variant: 'destructive',
          });
        },
      }
    );
  };

  const handleToggleAll = (enabled: boolean) => {
    toggleAllEndpoints.mutate(enabled, {
      onSuccess: () => {
        toast({
          title: enabled ? 'All endpoints enabled' : 'All endpoints disabled',
        });
      },
      onError: (err) => {
        toast({
          title: 'Failed to toggle endpoints',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  };

  const enabledCount = endpoints?.filter((e) => e.enabled).length || 0;
  const totalCount = endpoints?.length || 0;

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Outgoing Endpoints</h1>
          <p className="text-sm text-muted-foreground">
            Configure HTTP endpoints for load testing
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => handleToggleAll(true)}>
            <Power className="h-4 w-4 mr-1" />
            Enable All
          </Button>
          <Button variant="outline" size="sm" onClick={() => handleToggleAll(false)}>
            <PowerOff className="h-4 w-4 mr-1" />
            Disable All
          </Button>
          <Button onClick={() => setCreateDialogOpen(true)}>
            <Plus className="h-4 w-4 mr-1" />
            Add Endpoint
          </Button>
        </div>
      </div>

      {/* Endpoints Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Send className="h-4 w-4" />
            Endpoints
            <Badge variant="secondary" className="ml-auto">
              {enabledCount}/{totalCount} active
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-4 space-y-4">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-12 w-full" />
              ))}
            </div>
          ) : !endpoints || endpoints.length === 0 ? (
            <div className="p-8 text-center">
              <Send className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">No endpoints configured</p>
              <Button
                variant="outline"
                className="mt-4"
                onClick={() => setCreateDialogOpen(true)}
              >
                <Plus className="h-4 w-4 mr-1" />
                Add your first endpoint
              </Button>
            </div>
          ) : (
            <ScrollArea className="h-[500px]">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-12">Active</TableHead>
                    <TableHead>Name</TableHead>
                    <TableHead>Method</TableHead>
                    <TableHead>URL</TableHead>
                    <TableHead className="text-right">Freq/min</TableHead>
                    <TableHead>Auth</TableHead>
                    <TableHead className="w-24">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {endpoints.map((endpoint) => (
                    <TableRow key={endpoint.name}>
                      <TableCell>
                        <Switch
                          checked={endpoint.enabled}
                          onCheckedChange={(checked) =>
                            handleToggle(endpoint.name, checked)
                          }
                        />
                      </TableCell>
                      <TableCell className="font-medium">
                        {endpoint.name}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant="outline"
                          className={cn(getMethodColor(endpoint.method))}
                        >
                          {endpoint.method}
                        </Badge>
                      </TableCell>
                      <TableCell className="max-w-[300px]">
                        <span className="truncate block text-muted-foreground text-xs">
                          {truncateUrl(endpoint.url_template, 60)}
                        </span>
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {endpoint.frequency}
                      </TableCell>
                      <TableCell>
                        {endpoint.auth ? (
                          <Badge variant="secondary">
                            {typeof endpoint.auth === 'string'
                              ? endpoint.auth
                              : (endpoint.auth as { ref?: string }).ref || 'inline'}
                          </Badge>
                        ) : (
                          <span className="text-muted-foreground text-xs">none</span>
                        )}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setEditingEndpoint(endpoint)}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setDeleteConfirmEndpoint(endpoint.name)}
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
            <DialogTitle>Create Endpoint</DialogTitle>
            <DialogDescription>
              Add a new outgoing HTTP endpoint for load testing
            </DialogDescription>
          </DialogHeader>
          <EndpointForm
            onSubmit={handleCreate}
            onCancel={() => setCreateDialogOpen(false)}
            isLoading={createEndpoint.isPending}
          />
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editingEndpoint} onOpenChange={() => setEditingEndpoint(null)}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Endpoint</DialogTitle>
            <DialogDescription>
              Modify endpoint configuration
            </DialogDescription>
          </DialogHeader>
          {editingEndpoint && (
            <EndpointForm
              initialData={editingEndpoint}
              onSubmit={handleUpdate}
              onCancel={() => setEditingEndpoint(null)}
              isLoading={updateEndpoint.isPending}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={!!deleteConfirmEndpoint}
        onOpenChange={() => setDeleteConfirmEndpoint(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Endpoint</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete "{deleteConfirmEndpoint}"? This action
              cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirmEndpoint(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteEndpoint.isPending}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
