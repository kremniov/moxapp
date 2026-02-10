import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import type { AuthConfig, AuthConfigRequest, AuthType } from '@/types/api';

interface AuthFormProps {
  initialData?: AuthConfig;
  onSubmit: (data: AuthConfigRequest) => void;
  onCancel: () => void;
  isLoading?: boolean;
}

const AUTH_TYPES: { value: AuthType; label: string; description: string }[] = [
  { value: 'none', label: 'None', description: 'No authentication' },
  { value: 'bearer', label: 'Bearer Token', description: 'Authorization: Bearer <token>' },
  { value: 'api_key', label: 'API Key (Header)', description: 'Custom header with API key' },
  { value: 'api_key_query', label: 'API Key (Query)', description: 'Query parameter with API key' },
  { value: 'basic', label: 'Basic Auth', description: 'HTTP Basic authentication' },
  { value: 'custom_header', label: 'Custom Header', description: 'Custom header with value' },
];

export function AuthForm({
  initialData,
  onSubmit,
  onCancel,
  isLoading,
}: AuthFormProps) {
  const [formData, setFormData] = useState({
    name: initialData?.name || '',
    type: initialData?.type || 'bearer' as AuthType,
    description: initialData?.description || '',
    header_name: initialData?.header_name || '',
    query_param: initialData?.query_param || '',
    env_var: initialData?.env_var || '',
    username_env: initialData?.username_env || '',
    password_env: initialData?.password_env || '',
    refresh_before_expiry: initialData?.refresh_before_expiry || 60,
    // Token endpoint fields
    token_url_env: initialData?.token_endpoint?.url_env || '',
    token_method: initialData?.token_endpoint?.method || 'POST',
    token_path: initialData?.token_endpoint?.token_path || 'access_token',
    expires_path: initialData?.token_endpoint?.expires_path || 'expires_in',
    token_headers: initialData?.token_endpoint?.headers
      ? JSON.stringify(initialData.token_endpoint.headers, null, 2)
      : '{}',
    token_body: initialData?.token_endpoint?.body
      ? JSON.stringify(initialData.token_endpoint.body, null, 2)
      : '',
  });

  const [errors, setErrors] = useState<Record<string, string>>({});
  const [hasTokenEndpoint, setHasTokenEndpoint] = useState(!!initialData?.token_endpoint);

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    } else if (!/^[a-z][a-z0-9_]*$/i.test(formData.name)) {
      newErrors.name = 'Name must start with a letter and contain only letters, numbers, and underscores';
    }

    // Type-specific validation
    if (formData.type === 'bearer' && !hasTokenEndpoint && !formData.env_var) {
      newErrors.env_var = 'Environment variable is required for static bearer tokens';
    }

    if (formData.type === 'api_key' && !formData.header_name) {
      newErrors.header_name = 'Header name is required';
    }

    if (formData.type === 'api_key_query' && !formData.query_param) {
      newErrors.query_param = 'Query parameter name is required';
    }

    if (formData.type === 'basic') {
      if (!formData.username_env) newErrors.username_env = 'Username env var is required';
      if (!formData.password_env) newErrors.password_env = 'Password env var is required';
    }

    if (formData.type === 'custom_header' && !formData.header_name) {
      newErrors.header_name = 'Header name is required';
    }

    if (hasTokenEndpoint) {
      if (!formData.token_url_env) {
        newErrors.token_url_env = 'Token URL env var is required';
      }
      if (formData.token_headers) {
        try {
          JSON.parse(formData.token_headers);
        } catch {
          newErrors.token_headers = 'Invalid JSON';
        }
      }
      if (formData.token_body) {
        try {
          JSON.parse(formData.token_body);
        } catch {
          newErrors.token_body = 'Invalid JSON';
        }
      }
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    const data: AuthConfigRequest = {
      name: formData.name.trim(),
      type: formData.type,
      description: formData.description || undefined,
    };

    // Add type-specific fields
    if (formData.type === 'bearer') {
      data.env_var = formData.env_var || undefined;
      if (hasTokenEndpoint) {
        data.token_endpoint = {
          url_env: formData.token_url_env,
          method: formData.token_method,
          token_path: formData.token_path,
          expires_path: formData.expires_path || undefined,
          headers: formData.token_headers ? JSON.parse(formData.token_headers) : undefined,
          body: formData.token_body ? JSON.parse(formData.token_body) : undefined,
        };
        data.refresh_before_expiry = formData.refresh_before_expiry;
      }
    }

    if (formData.type === 'api_key' || formData.type === 'custom_header') {
      data.header_name = formData.header_name;
      data.env_var = formData.env_var;
    }

    if (formData.type === 'api_key_query') {
      data.query_param = formData.query_param;
      data.env_var = formData.env_var;
    }

    if (formData.type === 'basic') {
      data.username_env = formData.username_env;
      data.password_env = formData.password_env;
    }

    onSubmit(data);
  };

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
            placeholder="my_auth_config"
            disabled={!!initialData}
          />
          {errors.name && (
            <p className="text-xs text-destructive">{errors.name}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="type">Type</Label>
          <Select
            value={formData.type}
            onValueChange={(value) => setFormData({ ...formData, type: value as AuthType })}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {AUTH_TYPES.map((type) => (
                <SelectItem key={type.value} value={type.value}>
                  <div>
                    <div>{type.label}</div>
                    <div className="text-xs text-muted-foreground">{type.description}</div>
                  </div>
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="description">Description (optional)</Label>
        <Input
          id="description"
          value={formData.description}
          onChange={(e) => setFormData({ ...formData, description: e.target.value })}
          placeholder="Brief description of this auth config"
        />
      </div>

      <Separator />

      {/* Type-specific fields */}
      {formData.type === 'bearer' && (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="env_var">Token Environment Variable</Label>
            <Input
              id="env_var"
              value={formData.env_var}
              onChange={(e) => setFormData({ ...formData, env_var: e.target.value })}
              placeholder="MY_BEARER_TOKEN"
              className="font-mono"
            />
            <p className="text-xs text-muted-foreground">
              For static tokens, or as fallback when token endpoint fails
            </p>
            {errors.env_var && (
              <p className="text-xs text-destructive">{errors.env_var}</p>
            )}
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="hasTokenEndpoint"
              checked={hasTokenEndpoint}
              onChange={(e) => setHasTokenEndpoint(e.target.checked)}
              className="rounded"
            />
            <Label htmlFor="hasTokenEndpoint" className="cursor-pointer">
              Configure token refresh endpoint
            </Label>
          </div>

          {hasTokenEndpoint && (
            <div className="space-y-4 pl-4 border-l-2 border-primary/20">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="token_url_env">Token URL Env Var</Label>
                  <Input
                    id="token_url_env"
                    value={formData.token_url_env}
                    onChange={(e) =>
                      setFormData({ ...formData, token_url_env: e.target.value })
                    }
                    placeholder="TOKEN_ENDPOINT_URL"
                    className="font-mono"
                  />
                  {errors.token_url_env && (
                    <p className="text-xs text-destructive">{errors.token_url_env}</p>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="token_method">Method</Label>
                  <Select
                    value={formData.token_method}
                    onValueChange={(value) =>
                      setFormData({ ...formData, token_method: value })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="POST">POST</SelectItem>
                      <SelectItem value="GET">GET</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="token_path">Token JSON Path</Label>
                  <Input
                    id="token_path"
                    value={formData.token_path}
                    onChange={(e) =>
                      setFormData({ ...formData, token_path: e.target.value })
                    }
                    placeholder="access_token"
                    className="font-mono"
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="expires_path">Expires JSON Path</Label>
                  <Input
                    id="expires_path"
                    value={formData.expires_path}
                    onChange={(e) =>
                      setFormData({ ...formData, expires_path: e.target.value })
                    }
                    placeholder="expires_in"
                    className="font-mono"
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="refresh_before_expiry">Refresh Before (sec)</Label>
                  <Input
                    id="refresh_before_expiry"
                    type="number"
                    min={0}
                    value={formData.refresh_before_expiry}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        refresh_before_expiry: parseInt(e.target.value, 10) || 60,
                      })
                    }
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="token_headers">Token Request Headers (JSON)</Label>
                <Textarea
                  id="token_headers"
                  value={formData.token_headers}
                  onChange={(e) =>
                    setFormData({ ...formData, token_headers: e.target.value })
                  }
                  placeholder='{"Content-Type": "application/json"}'
                  className="font-mono text-sm h-20"
                />
                {errors.token_headers && (
                  <p className="text-xs text-destructive">{errors.token_headers}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="token_body">Token Request Body (JSON)</Label>
                <Textarea
                  id="token_body"
                  value={formData.token_body}
                  onChange={(e) =>
                    setFormData({ ...formData, token_body: e.target.value })
                  }
                  placeholder='{"grant_type": "client_credentials"}'
                  className="font-mono text-sm h-24"
                />
                <p className="text-xs text-muted-foreground">
                  Use {'{{ env "VAR_NAME" }}'} for environment variables
                </p>
                {errors.token_body && (
                  <p className="text-xs text-destructive">{errors.token_body}</p>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {(formData.type === 'api_key' || formData.type === 'custom_header') && (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="header_name">Header Name</Label>
            <Input
              id="header_name"
              value={formData.header_name}
              onChange={(e) => setFormData({ ...formData, header_name: e.target.value })}
              placeholder="X-API-Key"
              className="font-mono"
            />
            {errors.header_name && (
              <p className="text-xs text-destructive">{errors.header_name}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="env_var">Value Environment Variable</Label>
            <Input
              id="env_var"
              value={formData.env_var}
              onChange={(e) => setFormData({ ...formData, env_var: e.target.value })}
              placeholder="MY_API_KEY"
              className="font-mono"
            />
          </div>
        </div>
      )}

      {formData.type === 'api_key_query' && (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="query_param">Query Parameter Name</Label>
            <Input
              id="query_param"
              value={formData.query_param}
              onChange={(e) => setFormData({ ...formData, query_param: e.target.value })}
              placeholder="api_key"
              className="font-mono"
            />
            {errors.query_param && (
              <p className="text-xs text-destructive">{errors.query_param}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="env_var">Value Environment Variable</Label>
            <Input
              id="env_var"
              value={formData.env_var}
              onChange={(e) => setFormData({ ...formData, env_var: e.target.value })}
              placeholder="MY_API_KEY"
              className="font-mono"
            />
          </div>
        </div>
      )}

      {formData.type === 'basic' && (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="username_env">Username Environment Variable</Label>
            <Input
              id="username_env"
              value={formData.username_env}
              onChange={(e) => setFormData({ ...formData, username_env: e.target.value })}
              placeholder="BASIC_AUTH_USER"
              className="font-mono"
            />
            {errors.username_env && (
              <p className="text-xs text-destructive">{errors.username_env}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="password_env">Password Environment Variable</Label>
            <Input
              id="password_env"
              value={formData.password_env}
              onChange={(e) => setFormData({ ...formData, password_env: e.target.value })}
              placeholder="BASIC_AUTH_PASS"
              className="font-mono"
            />
            {errors.password_env && (
              <p className="text-xs text-destructive">{errors.password_env}</p>
            )}
          </div>
        </div>
      )}

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
