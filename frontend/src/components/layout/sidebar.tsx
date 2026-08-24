import { useEffect, useState } from 'react';
import { usePathname } from '@/lib/navigation';
import { Link } from '@/lib/link';
import {
  Inbox,
  Radio,
  Send,
  Library,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  ChevronDown,
  BookOpen,
  Hash,
  Siren,
  MessagesSquare,
  Send as SendIcon,
  Bell,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { APP_VERSION } from '@/lib/env';
import { useUIStore } from '@/lib/store';
import { LogoMark } from '@/components/brand';

type NavItem = {
  label: string;
  href: string;
  icon: typeof Inbox;
  exact?: boolean;
};

type NavGroup = {
  label: string;
  items: NavItem[];
  defaultOpen?: boolean;
};

const navGroups: NavGroup[] = [
  {
    label: 'Capture',
    defaultOpen: true,
    items: [
      { label: 'Inbox', href: '/', icon: Inbox, exact: true },
      { label: 'Sinks', href: '/sinks', icon: Radio },
    ],
  },
  {
    label: 'Mocks',
    defaultOpen: true,
    items: [
      { label: 'Slack', href: '/slack', icon: Hash },
      { label: 'Teams', href: '/teams', icon: MessagesSquare },
      { label: 'Discord', href: '/discord', icon: Hash },
      { label: 'PagerDuty', href: '/pagerduty', icon: Siren },
      { label: 'Telegram', href: '/telegram', icon: SendIcon },
      { label: 'Google Chat', href: '/googlechat', icon: MessagesSquare },
      { label: 'Mattermost', href: '/mattermost', icon: Hash },
      { label: 'Opsgenie', href: '/opsgenie', icon: Bell },
    ],
  },
  {
    label: 'Test',
    defaultOpen: true,
    items: [
      { label: 'Send', href: '/send', icon: Send },
      { label: 'Fixtures', href: '/fixtures', icon: Library },
    ],
  },
];

function SidebarGroup({
  group,
  pathname,
  collapsed,
  isOpen,
  onToggle,
}: {
  group: NavGroup;
  pathname: string;
  collapsed: boolean;
  isOpen: boolean;
  onToggle: () => void;
}) {
  if (collapsed) {
    return (
      <div className="space-y-0.5">
        {group.items.map((item) => {
          const Icon = item.icon;
          const active = item.exact ? pathname === item.href : pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn('nav-item group justify-center px-0', active && 'active')}
              title={item.label}
            >
              <Icon
                className={cn(
                  'h-4 w-4 flex-shrink-0',
                  active ? 'text-foreground' : 'text-muted-foreground group-hover:text-foreground',
                )}
              />
            </Link>
          );
        })}
      </div>
    );
  }

  return (
    <div>
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between px-3 py-2 text-sm font-semibold text-muted-foreground hover:text-foreground transition-colors"
      >
        <span>{group.label}</span>
        {isOpen ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
      </button>
      {isOpen && (
        <div className="space-y-px">
          {group.items.map((item) => {
            const Icon = item.icon;
            const active = item.exact ? pathname === item.href : pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  'flex items-center gap-2 px-3 py-1.5 mx-1 rounded-md text-sm transition-colors',
                  active
                    ? 'bg-accent text-foreground font-medium'
                    : 'text-muted-foreground hover:text-foreground hover:bg-accent/50',
                )}
              >
                <Icon
                  className={cn(
                    'h-4 w-4 flex-shrink-0',
                    active ? 'text-foreground' : 'text-muted-foreground',
                  )}
                />
                <span className="truncate flex-1">{item.label}</span>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}

export function Sidebar() {
  const pathname = usePathname();
  const { sidebarCollapsed, toggleSidebarCollapsed } = useUIStore();
  const [openGroups, setOpenGroups] = useState<Set<string>>(
    () => new Set(navGroups.filter((g) => g.defaultOpen).map((g) => g.label)),
  );

  useEffect(() => {
    setOpenGroups((prev) => {
      const next = new Set(prev);
      let changed = false;
      const activeGroup = navGroups.find((g) =>
        g.items.some((item) => (item.exact ? pathname === item.href : pathname.startsWith(item.href))),
      );
      if (activeGroup && !next.has(activeGroup.label)) {
        next.add(activeGroup.label);
        changed = true;
      }
      return changed ? next : prev;
    });
  }, [pathname]);

  return (
    <aside
      className={cn(
        'flex flex-col h-screen bg-sidebar border-r border-sidebar-border transition-all duration-200 ease-in-out',
        sidebarCollapsed ? 'w-16' : 'w-60',
      )}
    >
      <div className="flex items-center h-14 px-4 border-b border-sidebar-border">
        <Link href="/" className="flex items-center gap-2.5 min-w-0" title="Webhookie">
          <LogoMark />
          {!sidebarCollapsed && (
            <div className="flex flex-col min-w-0">
              <span className="text-sm font-semibold text-foreground tracking-tight truncate leading-tight">
                Webhookie
              </span>
              <span className="text-[10px] text-muted-foreground leading-tight">by AlphaBravo</span>
            </div>
          )}
        </Link>
        <button
          onClick={toggleSidebarCollapsed}
          className={cn('nav-item', sidebarCollapsed ? 'hidden' : 'ml-auto')}
          title="Collapse sidebar"
          aria-label="Collapse sidebar"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>
      </div>
      {sidebarCollapsed && (
        <button
          onClick={toggleSidebarCollapsed}
          className="nav-item mx-1 mt-1 justify-center"
          title="Expand sidebar"
          aria-label="Expand sidebar"
        >
          <ChevronRight className="h-4 w-4" />
        </button>
      )}

      <nav className="flex-1 overflow-y-auto py-2 px-1 no-scrollbar">
        {navGroups.map((group) => (
          <SidebarGroup
            key={group.label}
            group={group}
            pathname={pathname}
            collapsed={sidebarCollapsed}
            isOpen={openGroups.has(group.label)}
            onToggle={() =>
              setOpenGroups((prev) => {
                const next = new Set(prev);
                if (next.has(group.label)) next.delete(group.label);
                else next.add(group.label);
                return next;
              })
            }
          />
        ))}
      </nav>

      <div className="mt-auto px-2 py-2 border-t border-sidebar-border space-y-1">
        <a
          href="https://github.com/alphabravo-oss/webhookie"
          target="_blank"
          rel="noopener noreferrer"
          className="nav-item w-full"
          title="Documentation"
        >
          <BookOpen className="h-4 w-4" />
          {!sidebarCollapsed && <span className="text-xs">Documentation</span>}
        </a>
        {!sidebarCollapsed && (
          <div className="px-3 py-1 space-y-0.5">
            <p className="text-[10px] text-muted-foreground">Webhookie {APP_VERSION}</p>
            <p className="text-[10px] text-muted-foreground">
              Built by{' '}
              <a
                href="https://alphabravo.io"
                target="_blank"
                rel="noopener noreferrer"
                className="hover:text-foreground underline-offset-2 hover:underline"
              >
                AlphaBravo
              </a>
            </p>
          </div>
        )}
      </div>
    </aside>
  );
}
