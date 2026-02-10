import { lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TooltipProvider } from '@/components/ui/tooltip';
import { Toaster } from '@/components/ui/toaster';
import { ThemeProvider } from '@/hooks/use-theme';
import { Shell } from '@/components/layout/shell';
import { Skeleton } from '@/components/ui/skeleton';

// Lazy load pages for code splitting
const DashboardPage = lazy(() => import('@/pages/dashboard').then(m => ({ default: m.DashboardPage })));
const AdminEndpointsPage = lazy(() => import('@/pages/admin-endpoints').then(m => ({ default: m.AdminEndpointsPage })));
const AdminRoutesPage = lazy(() => import('@/pages/admin-routes').then(m => ({ default: m.AdminRoutesPage })));
const AdminAuthPage = lazy(() => import('@/pages/admin-auth').then(m => ({ default: m.AdminAuthPage })));
const AdminConfigPage = lazy(() => import('@/pages/admin-config').then(m => ({ default: m.AdminConfigPage })));

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

function PageLoader() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-48" />
      <div className="grid gap-4 grid-cols-3">
        <Skeleton className="h-32" />
        <Skeleton className="h-32" />
        <Skeleton className="h-32" />
      </div>
      <Skeleton className="h-64" />
    </div>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <TooltipProvider delayDuration={300}>
          <BrowserRouter>
            <Routes>
              <Route element={<Shell />}>
                <Route
                  path="/"
                  element={
                    <Suspense fallback={<PageLoader />}>
                      <DashboardPage />
                    </Suspense>
                  }
                />
                <Route
                  path="/admin/endpoints"
                  element={
                    <Suspense fallback={<PageLoader />}>
                      <AdminEndpointsPage />
                    </Suspense>
                  }
                />
                <Route
                  path="/admin/routes"
                  element={
                    <Suspense fallback={<PageLoader />}>
                      <AdminRoutesPage />
                    </Suspense>
                  }
                />
                <Route
                  path="/admin/auth"
                  element={
                    <Suspense fallback={<PageLoader />}>
                      <AdminAuthPage />
                    </Suspense>
                  }
                />
                <Route
                  path="/admin/config"
                  element={
                    <Suspense fallback={<PageLoader />}>
                      <AdminConfigPage />
                    </Suspense>
                  }
                />
              </Route>
            </Routes>
          </BrowserRouter>
          <Toaster />
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
