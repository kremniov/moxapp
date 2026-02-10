import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import type { OutgoingEndpoint, OutgoingEndpointRequest } from '@/types/api';
import { useAuthConfigs } from '@/hooks/use-auth-configs';

interface EndpointFormProps {
  initialData?: OutgoingEndpoint;
  onSubmit: (data: OutgoingEndpointRequest) => void;
  onCancel: () => void;
  isLoading?: boolean;
}

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];

export function EndpointForm({
  initialData,
  onSubmit,
  onCancel,
  isLoading,
}: EndpointFormProps) {
  const { data: authConfigs } = useAuthConfigs();

  const [formData, setFormData] = useState({
    name: initialData?.name || '',
    method: initialData?.method || 'GET',
    url_template: initialData?.url_template || '',
    frequency: initialData?.frequency || 10,
    auth: typeof initialData?.auth === 'string' ? initialData.auth : 'none',
    timeout: initialData?.timeout || 30,
    enabled: initialData?.enabled ?? true,
    headers: JSON.stringify(initialData?.headers || {}, null, 2),
    body: initialData?.body ? JSON.stringify(initialData.body, null, 2) : '',
  });

  const [errors, setErrors] = useState<Record<string, string>>({});

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    } else if (!/^[a-z][a-z0-9_-]*$/i.test(formData.name)) {
      newErrors.name = 'Name must start with a letter and contain only letters, numbers, underscores, and hyphens';
    }

    if (!formData.url_template.trim()) {
      newErrors.url_template = 'URL template is required';
    }

    if (formData.frequency <= 0) {
      newErrors.frequency = 'Frequency must be greater than 0';
    }

    if (formData.headers) {
      try {
        JSON.parse(formData.headers);
      } catch {
        newErrors.headers = 'Invalid JSON';
      }
    }

    if (formData.body) {
      try {
        JSON.parse(formData.body);
      } catch {
        newErrors.body = 'Invalid JSON';
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    const data: OutgoingEndpointRequest = {
      name: formData.name.trim(),
      method: formData.method,
      url_template: formData.url_template.trim(),
      frequency: formData.frequency,
      auth: formData.auth === 'none' ? undefined : formData.auth,
      timeout: formData.timeout,
      enabled: formData.enabled,
      headers: formData.headers ? JSON.parse(formData.headers) : undefined,
      body: formData.body ? JSON.parse(formData.body) : undefined,
    };

    onSubmit(data);
  };

  const needsBody = ['POST', 'PUT', 'PATCH'].includes(formData.method);

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Basic Info */}
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="name">Name</Label>
          <Input
            id="name"
            value={formData.name}
            onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            placeholder="my_endpoint"
            disabled={!!initialData}
          />
          {errors.name && (
            <p className="text-xs text-destructive">{errors.name}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="method">Method</Label>
          <Select
            value={formData.method}
            onValueChange={(value) => setFormData({ ...formData, method: value as 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH' | 'HEAD' | 'OPTIONS' })}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {HTTP_METHODS.map((method) => (
                <SelectItem key={method} value={method}>
                  {method}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* URL Template */}
      <div className="space-y-2">
        <Label htmlFor="url_template">URL Template</Label>
        <Input
          id="url_template"
          value={formData.url_template}
          onChange={(e) =>
            setFormData({ ...formData, url_template: e.target.value })
          }
          placeholder="https://api.example.com/endpoint"
          className="font-mono text-sm"
        />
        <p className="text-xs text-muted-foreground">
          Supports Go templates: {'{{ .Env.VAR }}'}, {'{{ randomUUID }}'}, etc.
        </p>
        {errors.url_template && (
          <p className="text-xs text-destructive">{errors.url_template}</p>
        )}
      </div>

      <Separator />

      {/* Frequency and Timeout */}
      <div className="grid grid-cols-3 gap-4">
        <div className="space-y-2">
          <Label htmlFor="frequency">Frequency (req/min)</Label>
          <Input
            id="frequency"
            type="number"
            min={0.1}
            step={0.1}
            value={formData.frequency}
            onChange={(e) =>
              setFormData({ ...formData, frequency: parseFloat(e.target.value) || 0 })
            }
          />
          {errors.frequency && (
            <p className="text-xs text-destructive">{errors.frequency}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="timeout">Timeout (seconds)</Label>
          <Input
            id="timeout"
            type="number"
            min={1}
            value={formData.timeout}
            onChange={(e) =>
              setFormData({ ...formData, timeout: parseInt(e.target.value, 10) || 30 })
            }
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="auth">Authentication</Label>
          <Select
            value={formData.auth}
            onValueChange={(value) => setFormData({ ...formData, auth: value })}
          >
            <SelectTrigger>
              <SelectValue placeholder="None" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="none">None</SelectItem>
              {authConfigs?.map((config) => (
                <SelectItem key={config.name} value={config.name}>
                  {config.name} ({config.type})
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <Separator />

      {/* Headers */}
      <div className="space-y-2">
        <Label htmlFor="headers">Custom Headers (JSON)</Label>
        <Textarea
          id="headers"
          value={formData.headers}
          onChange={(e) => setFormData({ ...formData, headers: e.target.value })}
          placeholder='{"X-Custom-Header": "value"}'
          className="font-mono text-sm h-24"
        />
        {errors.headers && (
          <p className="text-xs text-destructive">{errors.headers}</p>
        )}
      </div>

      {/* Body (for POST/PUT/PATCH) */}
      {needsBody && (
        <div className="space-y-2">
          <Label htmlFor="body">Request Body (JSON)</Label>
          <Textarea
            id="body"
            value={formData.body}
            onChange={(e) => setFormData({ ...formData, body: e.target.value })}
            placeholder='{"key": "value"}'
            className="font-mono text-sm h-32"
          />
          <p className="text-xs text-muted-foreground">
            Supports templates in string values
          </p>
          {errors.body && (
            <p className="text-xs text-destructive">{errors.body}</p>
          )}
        </div>
      )}

      <Separator />

      {/* Enabled */}
      <div className="flex items-center justify-between">
        <div>
          <Label>Enabled</Label>
          <p className="text-xs text-muted-foreground">
            Enable this endpoint for load testing
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
