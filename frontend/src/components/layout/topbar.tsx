import { usePathname, useRouter } from '@/lib/navigation';
import { useEffect, useState } from 'react';
import { useTheme } from '@/lib/theme';
import { ChevronRight, Command, Sun, Moon, Monitor } from 'lucide-react';
import { useUIStore } from '@/lib/store';

const routeLabels: Record<string, string> = {
  sinks: 'Sinks',
  send: 'Send',
  fixtures: 'Fixtures',
  slack: 'Slack',
  teams: 'Teams',
  discord: 'Discord',
  pagerduty: 'PagerDuty',
  telegram: 'Telegram',
  googlechat: 'Google Chat',
  mattermost: 'Mattermost',
  opsgenie: 'Opsgenie',
};

function generateBreadcrumbs(pathname: string) {
  if (pathname === '/') return [{ label: 'Inbox', href: '/' }];
  const segments = pathname.split('/').filter(Boolean);
  const crumbs: { label: string; href: string }[] = [{ label: 'Inbox', href: '/' }];
  let path = '';
  for (const segment of segments) {
    path += `/${segment}`;
    crumbs.push({
      label: routeLabels[segment] || decodeURIComponent(segment),
      href: path,
    });
  }
  return crumbs;
}

export function Topbar() {
  const pathname = usePathname();
  const router = useRouter();
  const { setCommandPaletteOpen } = useUIStore();
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  const breadcrumbs = generateBreadcrumbs(pathname);

  useEffect(() => {
    setMounted(true);
  }, []);

  const cycleTheme = () => {
    if (theme === 'light') setTheme('dark');
    else if (theme === 'dark') setTheme('system');
    else setTheme('light');
  };

  const visibleTheme = mounted ? theme || 'system' : 'dark';
  const ThemeIcon = !mounted
    ? Moon
    : visibleTheme === 'dark'
      ? Moon
      : visibleTheme === 'light'
        ? Sun
        : Monitor;

  return (
    <header className="sticky top-0 z-30 flex items-center justify-between h-14 px-6 border-b border-border bg-background/80 backdrop-blur-lg">
      <nav className="flex items-center gap-1.5 text-sm min-w-0">
        {breadcrumbs.map((crumb, i) => (
          <div key={crumb.href} className="flex items-center gap-1.5 min-w-0">
            {i > 0 && <ChevronRight className="h-3.5 w-3.5 text-muted-foreground flex-shrink-0" />}
            {i === breadcrumbs.length - 1 ? (
              <span className="text-foreground font-medium truncate">{crumb.label}</span>
            ) : (
              <button
                onClick={() => router.push(crumb.href)}
                className="text-muted-foreground hover:text-foreground transition-colors truncate"
              >
                {crumb.label}
              </button>
            )}
          </div>
        ))}
      </nav>

      <div className="flex items-center gap-2">
        <button
          onClick={() => setCommandPaletteOpen(true)}
          className="inline-flex items-center gap-1.5 h-8 px-2.5 rounded-md border border-border text-xs
            text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
        >
          <Command className="h-3.5 w-3.5" />
          <kbd className="font-mono text-[10px]">K</kbd>
        </button>

        <button
          onClick={cycleTheme}
          className="relative inline-flex items-center justify-center h-8 w-8 rounded-md
            text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
          title={`Theme: ${visibleTheme}`}
          aria-label={`Theme: ${visibleTheme}`}
        >
          <ThemeIcon className="h-4 w-4" />
        </button>
      </div>
    </header>
  );
}
