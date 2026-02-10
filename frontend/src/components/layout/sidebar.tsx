import { NavLink, useLocation } from 'react-router-dom';
import {
  LayoutDashboard,
  Send,
  Inbox,
  KeyRound,
  FileText,
  ChevronDown,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useState } from 'react';

const navItems = [
  {
    title: 'Dashboard',
    href: '/',
    icon: LayoutDashboard,
  },
  {
    title: 'Admin',
    icon: ChevronDown,
    children: [
      {
        title: 'Endpoints',
        href: '/admin/endpoints',
        icon: Send,
        description: 'Outgoing HTTP endpoints',
      },
      {
        title: 'Routes',
        href: '/admin/routes',
        icon: Inbox,
        description: 'Incoming simulated routes',
      },
      {
        title: 'Auth',
        href: '/admin/auth',
        icon: KeyRound,
        description: 'Authentication configs',
      },
      {
        title: 'Config',
        href: '/admin/config',
        icon: FileText,
        description: 'Import/export YAML config',
      },
    ],
  },
];

export function Sidebar() {
  const location = useLocation();
  const [adminExpanded, setAdminExpanded] = useState(
    location.pathname.startsWith('/admin')
  );

  return (
    <aside className="flex w-56 flex-col border-r bg-card">
      {/* Logo */}
      <NavLink
        to="/"
        className="flex h-14 items-center border-b px-4 hover:bg-accent/40 transition-colors"
      >
        <div className="flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded bg-primary text-primary-foreground font-mono font-bold text-sm">
            MX
          </div>
          <span className="font-mono text-sm font-semibold tracking-tight">
            MOXAPP
          </span>
        </div>
      </NavLink>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 p-3">
        {navItems.map((item) =>
          item.children ? (
            <div key={item.title}>
              <button
                onClick={() => setAdminExpanded(!adminExpanded)}
                className={cn(
                  "flex w-full items-center justify-between rounded-md px-3 py-2 text-sm font-mono transition-colors",
                  "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                  adminExpanded && "text-foreground"
                )}
              >
                <span className="uppercase tracking-wider text-xs">
                  {item.title}
                </span>
                <ChevronDown
                  className={cn(
                    "h-4 w-4 transition-transform",
                    adminExpanded && "rotate-180"
                  )}
                />
              </button>
              {adminExpanded && (
                <div className="ml-2 mt-1 space-y-1 border-l border-border pl-3">
                  {item.children.map((child) => (
                    <NavLink
                      key={child.href}
                      to={child.href}
                      className={({ isActive }) =>
                        cn(
                          "flex items-center gap-2 rounded-md px-3 py-2 text-sm font-mono transition-colors",
                          isActive
                            ? "bg-primary/10 text-primary"
                            : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                        )
                      }
                    >
                      <child.icon className="h-4 w-4" />
                      <span>{child.title}</span>
                    </NavLink>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <NavLink
              key={item.href}
              to={item.href}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-2 rounded-md px-3 py-2 text-sm font-mono transition-colors",
                  isActive
                    ? "bg-primary/10 text-primary"
                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                )
              }
            >
              <item.icon className="h-4 w-4" />
              <span>{item.title}</span>
            </NavLink>
          )
        )}
      </nav>

      {/* Footer */}
      <div className="border-t p-3">
        <div className="rounded-md bg-muted/50 px-3 py-2">
          <p className="text-xs font-mono text-muted-foreground">
            DNS Load Test Tool
          </p>
          <p className="text-xs font-mono text-muted-foreground/60">v1.0.0</p>
        </div>
      </div>
    </aside>
  );
}
