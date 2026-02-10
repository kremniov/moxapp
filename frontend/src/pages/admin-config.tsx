import { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { useToast } from '@/hooks/use-toast';
import { configApi } from '@/lib/api';

export function AdminConfigPage() {
  const { toast } = useToast();
  const [yamlText, setYamlText] = useState('');
  const [isExporting, setIsExporting] = useState(false);
  const [isImporting, setIsImporting] = useState(false);

  const handleExport = async () => {
    try {
      setIsExporting(true);
      const content = await configApi.exportYaml();
      const blob = new Blob([content], { type: 'application/x-yaml' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'moxapp-config.yaml';
      link.click();
      URL.revokeObjectURL(url);
      toast({ title: 'Config exported', description: 'Download should start shortly.' });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Export failed';
      toast({ title: 'Export failed', description: message, variant: 'destructive' });
    } finally {
      setIsExporting(false);
    }
  };

  const handleFileUpload = async (file: File) => {
    try {
      setIsImporting(true);
      const content = await file.text();
      setYamlText(content);
      await configApi.importYaml(content);
      toast({ title: 'Config imported', description: 'In-memory config replaced.' });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Import failed';
      toast({ title: 'Import failed', description: message, variant: 'destructive' });
    } finally {
      setIsImporting(false);
    }
  };

  const handleImportText = async () => {
    if (!yamlText.trim()) {
      toast({ title: 'No YAML provided', description: 'Paste YAML to import.' });
      return;
    }

    try {
      setIsImporting(true);
      await configApi.importYaml(yamlText);
      toast({ title: 'Config imported', description: 'In-memory config replaced.' });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Import failed';
      toast({ title: 'Import failed', description: message, variant: 'destructive' });
    } finally {
      setIsImporting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-mono font-semibold">Config Import/Export</h1>
        <p className="text-muted-foreground">
          Download the full in-memory YAML config or replace it with a new file.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="font-mono">Export</CardTitle>
          <CardDescription>Download current in-memory configuration.</CardDescription>
        </CardHeader>
        <CardContent>
          <Button onClick={handleExport} disabled={isExporting}>
            {isExporting ? 'Exporting...' : 'Download YAML'}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="font-mono">Import</CardTitle>
          <CardDescription>
            Upload YAML to replace the in-memory configuration. Invalid files are rejected.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Input
            type="file"
            accept=".yaml,.yml,application/x-yaml,text/yaml"
            disabled={isImporting}
            onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) {
                handleFileUpload(file);
              }
            }}
          />

          <Textarea
            value={yamlText}
            onChange={(event) => setYamlText(event.target.value)}
            placeholder="Paste YAML here"
            className="min-h-[220px] font-mono text-xs"
          />

          <Button onClick={handleImportText} disabled={isImporting}>
            {isImporting ? 'Importing...' : 'Import from Paste'}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
