import { createRootRoute, Outlet } from '@tanstack/react-router';
import { Compass } from 'lucide-react';
import { Providers } from '@/components/providers';
import { Sidebar } from '@/components/layout/sidebar';
import { Topbar } from '@/components/layout/topbar';
import { CommandPalette } from '@/components/layout/command-palette';
import { StatePanel } from '@/components/ui/empty-state';

function NotFound() {
  return (
    <div data-testid="route-not-found" className="flex min-h-screen flex-col items-center justify-center px-6">
      <StatePanel
        icon={Compass}
        tone="info"
        title="404 — Page not found"
        description="The page you're looking for doesn't exist or has moved."
        actionLabel="Back to inbox"
        actionHref="/"
      />
    </div>
  );
}

function AppShell() {
  return (
    <div data-testid="app-shell" className="flex h-screen overflow-hidden bg-background">
      <Sidebar />
      <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
        <Topbar />
        <main className="flex-1 min-h-0 overflow-y-auto">
          <div className="p-6 max-w-[1600px] mx-auto animate-fade-in">
            <Outlet />
          </div>
        </main>
      </div>
      <CommandPalette />
    </div>
  );
}

export const Route = createRootRoute({
  component: () => (
    <Providers>
      <AppShell />
    </Providers>
  ),
  notFoundComponent: NotFound,
});
