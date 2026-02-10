import { useState } from 'react';
import {
  useRoutes,
  useCreateRoute,
  useUpdateRoute,
  useDeleteRoute,
  useToggleRoute,
  useToggleAllRoutes,
} from '@/hooks/use-routes';
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
  Inbox,
  Power,
  PowerOff,
} from 'lucide-react';
import { cn, getMethodColor } from '@/lib/utils';
import { RouteForm } from '@/components/admin/route-form';
import type { IncomingRoute, IncomingRouteRequest } from '@/types/api';

export function AdminRoutesPage() {
  const { data: routes, isLoading } = useRoutes();
  const createRoute = useCreateRoute();
  const updateRoute = useUpdateRoute();
  const deleteRoute = useDeleteRoute();
  const toggleRoute = useToggleRoute();
  const toggleAllRoutes = useToggleAllRoutes();
  const { toast } = useToast();

  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [editingRoute, setEditingRoute] = useState<IncomingRoute | null>(null);
  const [deleteConfirmRoute, setDeleteConfirmRoute] = useState<string | null>(null);

  const handleCreate = (data: IncomingRouteRequest) => {
    createRoute.mutate(data, {
      onSuccess: () => {
        setCreateDialogOpen(false);
        toast({ title: 'Route created', description: `${data.name} has been created` });
      },
      onError: (err) => {
        toast({
          title: 'Failed to create route',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  };

  const handleUpdate = (data: IncomingRouteRequest) => {
    if (!editingRoute) return;
    updateRoute.mutate(
      { name: editingRoute.name, data },
      {
        onSuccess: () => {
          setEditingRoute(null);
          toast({ title: 'Route updated', description: `${data.name} has been updated` });
        },
        onError: (err) => {
          toast({
            title: 'Failed to update route',
            description: err.message,
            variant: 'destructive',
          });
        },
      }
    );
  };

  const handleDelete = () => {
    if (!deleteConfirmRoute) return;
    deleteRoute.mutate(deleteConfirmRoute, {
      onSuccess: () => {
        setDeleteConfirmRoute(null);
        toast({ title: 'Route deleted' });
      },
      onError: (err) => {
        toast({
          title: 'Failed to delete route',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  };

  const handleToggle = (name: string, enabled: boolean) => {
    toggleRoute.mutate(
      { name, enabled },
      {
        onError: (err) => {
          toast({
            title: 'Failed to toggle route',
            description: err.message,
            variant: 'destructive',
          });
        },
      }
    );
  };

  const handleToggleAll = (enabled: boolean) => {
    toggleAllRoutes.mutate(enabled, {
      onSuccess: () => {
        toast({
          title: enabled ? 'All routes enabled' : 'All routes disabled',
        });
      },
      onError: (err) => {
        toast({
          title: 'Failed to toggle routes',
          description: err.message,
          variant: 'destructive',
        });
      },
    });
  };

  const enabledCount = routes?.filter((r) => r.enabled).length || 0;
  const totalCount = routes?.length || 0;

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Incoming Routes</h1>
          <p className="text-sm text-muted-foreground">
            Configure simulated incoming routes with response patterns
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
            Add Route
          </Button>
        </div>
      </div>

      {/* Routes Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Inbox className="h-4 w-4" />
            Routes
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
          ) : !routes || routes.length === 0 ? (
            <div className="p-8 text-center">
              <Inbox className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">No routes configured</p>
              <Button
                variant="outline"
                className="mt-4"
                onClick={() => setCreateDialogOpen(true)}
              >
                <Plus className="h-4 w-4 mr-1" />
                Add your first route
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
                    <TableHead>Path</TableHead>
                    <TableHead>Responses</TableHead>
                    <TableHead className="w-24">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {routes.map((route) => (
                    <TableRow key={route.name}>
                      <TableCell>
                        <Switch
                          checked={route.enabled}
                          onCheckedChange={(checked) =>
                            handleToggle(route.name, checked)
                          }
                        />
                      </TableCell>
                      <TableCell className="font-medium">{route.name}</TableCell>
                      <TableCell>
                        <Badge
                          variant="outline"
                          className={cn(
                            route.method === '*'
                              ? 'text-muted-foreground'
                              : getMethodColor(route.method)
                          )}
                        >
                          {route.method === '*' ? 'ANY' : route.method}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-sm text-muted-foreground">
                        {route.path}
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-1 flex-wrap">
                          {route.responses.map((resp, idx) => {
                            const variant =
                              resp.status >= 200 && resp.status < 300
                                ? 'success'
                                : resp.status >= 400 && resp.status < 500
                                ? 'warning'
                                : resp.status >= 500
                                ? 'error'
                                : 'secondary';
                            return (
                              <Badge key={idx} variant={variant}>
                                {resp.status} ({(resp.share * 100).toFixed(0)}%)
                              </Badge>
                            );
                          })}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setEditingRoute(route)}
                          >
                            <Pencil className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => setDeleteConfirmRoute(route.name)}
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
            <DialogTitle>Create Route</DialogTitle>
            <DialogDescription>
              Add a new incoming route with response configuration
            </DialogDescription>
          </DialogHeader>
          <RouteForm
            onSubmit={handleCreate}
            onCancel={() => setCreateDialogOpen(false)}
            isLoading={createRoute.isPending}
          />
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editingRoute} onOpenChange={() => setEditingRoute(null)}>
        <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit Route</DialogTitle>
            <DialogDescription>Modify route configuration</DialogDescription>
          </DialogHeader>
          {editingRoute && (
            <RouteForm
              initialData={editingRoute}
              onSubmit={handleUpdate}
              onCancel={() => setEditingRoute(null)}
              isLoading={updateRoute.isPending}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog
        open={!!deleteConfirmRoute}
        onOpenChange={() => setDeleteConfirmRoute(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Route</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete "{deleteConfirmRoute}"? This action
              cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteConfirmRoute(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteRoute.isPending}
            >
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
