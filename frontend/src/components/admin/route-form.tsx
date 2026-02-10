import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { Card, CardContent } from '@/components/ui/card';
import { Plus, Trash2 } from 'lucide-react';
import type { IncomingRoute, IncomingRouteRequest, IncomingResponseConfig } from '@/types/api';

interface RouteFormProps {
  initialData?: IncomingRoute;
  onSubmit: (data: IncomingRouteRequest) => void;
  onCancel: () => void;
  isLoading?: boolean;
}

const HTTP_METHODS = ['*', 'GET', 'POST', 'PUT', 'DELETE', 'PATCH'];

const DEFAULT_RESPONSE: IncomingResponseConfig = {
  status: 200,
  share: 1,
  min_response_ms: 10,
  max_response_ms: 50,
};

export function RouteForm({
  initialData,
  onSubmit,
  onCancel,
  isLoading,
}: RouteFormProps) {
  const [formData, setFormData] = useState({
    name: initialData?.name || '',
    path: initialData?.path || '/',
    method: initialData?.method || '*',
    enabled: initialData?.enabled ?? true,
    responses: initialData?.responses || [{ ...DEFAULT_RESPONSE }],
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    } else if (!/^[a-z][a-z0-9_]*$/i.test(formData.name)) {
      newErrors.name = 'Name must start with a letter and contain only letters, numbers, and underscores';
    }

    if (!formData.path.trim()) {
      newErrors.path = 'Path is required';
    } else if (!formData.path.startsWith('/')) {
      newErrors.path = 'Path must start with /';
    }

    if (formData.responses.length === 0) {
      newErrors.responses = 'At least one response is required';
    }

    const totalShare = formData.responses.reduce((sum, r) => sum + r.share, 0);
    if (Math.abs(totalShare - 1) > 0.01) {
      newErrors.responses = `Response shares must sum to 1.0 (currently ${totalShare.toFixed(2)})`;
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    const data: IncomingRouteRequest = {
      name: formData.name.trim(),
      path: formData.path.trim(),
      method: formData.method,
      enabled: formData.enabled,
      responses: formData.responses,
    };

    onSubmit(data);
  };

  const addResponse = () => {
    setFormData({
      ...formData,
      responses: [...formData.responses, { ...DEFAULT_RESPONSE, share: 0 }],
    });
  };

  const removeResponse = (index: number) => {
    setFormData({
      ...formData,
      responses: formData.responses.filter((_, i) => i !== index),
    });
  };

  const updateResponse = (index: number, field: keyof IncomingResponseConfig, value: number) => {
    const newResponses = [...formData.responses];
    newResponses[index] = { ...newResponses[index], [field]: value };
    setFormData({ ...formData, responses: newResponses });
  };

  const normalizeShares = () => {
    const total = formData.responses.reduce((sum, r) => sum + r.share, 0);
    if (total === 0) return;
    
    const newResponses = formData.responses.map((r) => ({
      ...r,
      share: parseFloat((r.share / total).toFixed(2)),
    }));
    
    // Adjust last item to ensure exact sum of 1
    const newTotal = newResponses.reduce((sum, r) => sum + r.share, 0);
    if (newResponses.length > 0) {
      newResponses[newResponses.length - 1].share += parseFloat((1 - newTotal).toFixed(2));
    }
    
    setFormData({ ...formData, responses: newResponses });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Basic Info */}
      <div className="grid grid-cols-3 gap-4">
        <div className="space-y-2">
          <Label htmlFor="name">Name</Label>
          <Input
            id="name"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            placeholder="my_route"
            disabled={!!initialData}
          />
          {errors.name && (
            <p className="text-xs text-destructive">{errors.name}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="path">Path</Label>
          <Input
            id="path"
            value={formData.path}
            onChange={(e) => setFormData({ ...formData, path: e.target.value })}
            placeholder="/api/endpoint"
            className="font-mono"
          />
          {errors.path && (
            <p className="text-xs text-destructive">{errors.path}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="method">Method</Label>
          <Select
            value={formData.method}
            onValueChange={(value) => setFormData({ ...formData, method: value })}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {HTTP_METHODS.map((method) => (
                <SelectItem key={method} value={method}>
                  {method === '*' ? 'ANY' : method}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <Separator />

      {/* Responses */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <Label>Response Configurations</Label>
            <p className="text-xs text-muted-foreground">
              Define weighted response behaviors
            </p>
          </div>
          <div className="flex gap-2">
            <Button type="button" variant="outline" size="sm" onClick={normalizeShares}>
              Normalize
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={addResponse}>
              <Plus className="h-4 w-4 mr-1" />
              Add Response
            </Button>
          </div>
        </div>

        {errors.responses && (
          <p className="text-xs text-destructive">{errors.responses}</p>
        )}

        <div className="space-y-3">
          {formData.responses.map((response, index) => (
            <Card key={index}>
              <CardContent className="pt-4">
                <div className="grid grid-cols-5 gap-3 items-end">
                  <div className="space-y-1">
                    <Label className="text-xs">Status Code</Label>
                    <Input
                      type="number"
                      min={100}
                      max={599}
                      value={response.status}
                      onChange={(e) =>
                        updateResponse(index, 'status', parseInt(e.target.value, 10) || 200)
                      }
                    />
                  </div>

                  <div className="space-y-1">
                    <Label className="text-xs">Share (0-1)</Label>
                    <Input
                      type="number"
                      min={0}
                      max={1}
                      step={0.01}
                      value={response.share}
                      onChange={(e) =>
                        updateResponse(index, 'share', parseFloat(e.target.value) || 0)
                      }
                    />
                  </div>

                  <div className="space-y-1">
                    <Label className="text-xs">Min Delay (ms)</Label>
                    <Input
                      type="number"
                      min={0}
                      value={response.min_response_ms}
                      onChange={(e) =>
                        updateResponse(index, 'min_response_ms', parseInt(e.target.value, 10) || 0)
                      }
                    />
                  </div>

                  <div className="space-y-1">
                    <Label className="text-xs">Max Delay (ms)</Label>
                    <Input
                      type="number"
                      min={0}
                      value={response.max_response_ms}
                      onChange={(e) =>
                        updateResponse(index, 'max_response_ms', parseInt(e.target.value, 10) || 0)
                      }
                    />
                  </div>

                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={() => removeResponse(index)}
                    disabled={formData.responses.length <= 1}
                    className="text-destructive hover:text-destructive"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>

      <Separator />

      {/* Enabled */}
      <div className="flex items-center justify-between">
        <div>
          <Label>Enabled</Label>
          <p className="text-xs text-muted-foreground">
            Enable this route for incoming traffic simulation
          </p>
        </div>
        <Switch
          checked={formData.enabled}
          onCheckedChange={(checked) =>
            setFormData({ ...formData, enabled: checked })
          }
        />
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-2 pt-4">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={isLoading}>
          {isLoading ? 'Saving...' : initialData ? 'Update' : 'Create'}
        </Button>
      </div>
    </form>
  );
}
