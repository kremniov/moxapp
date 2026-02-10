import { Outlet } from 'react-router-dom';
import { Sidebar } from './sidebar';
import { Header } from './header';
import { ScrollArea } from '@/components/ui/scroll-area';

export function Shell() {
  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header />
        <ScrollArea className="flex-1">
          <main className="p-6">
            <Outlet />
          </main>
        </ScrollArea>
      </div>
    </div>
  );
}
