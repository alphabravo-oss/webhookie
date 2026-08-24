import { useCallback, useEffect, useState } from 'react';
import { useRouter } from '@/lib/navigation';
import { Command } from 'cmdk';
import { Inbox, Radio, Send, Library, Search, ArrowRight, Hash, MessagesSquare, Siren, Bell } from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useUIStore } from '@/lib/store';
import { OverlayShell } from '@/components/ui/overlay-shell';

const pages = [
  { name: 'Inbox', href: '/', icon: Inbox, description: 'Captured webhook events' },
  { name: 'Sinks', href: '/sinks', icon: Radio, description: 'Destination URLs to copy' },
  { name: 'Send', href: '/send', icon: Send, description: 'Fire a signed fixture' },
  { name: 'Fixtures', href: '/fixtures', icon: Library, description: 'Provider sample library' },
  { name: 'Slack', href: '/slack', icon: Hash, description: 'Mock Slack workspace' },
  { name: 'Teams', href: '/teams', icon: MessagesSquare, description: 'Mock Teams workspace' },
  { name: 'Discord', href: '/discord', icon: Hash, description: 'Mock Discord server' },
  { name: 'PagerDuty', href: '/pagerduty', icon: Siren, description: 'Mock PagerDuty services' },
  { name: 'Telegram', href: '/telegram', icon: Send, description: 'Mock Telegram bot chats' },
  { name: 'Google Chat', href: '/googlechat', icon: MessagesSquare, description: 'Mock Google Chat spaces' },
  { name: 'Mattermost', href: '/mattermost', icon: Hash, description: 'Mock Mattermost channels' },
  { name: 'Opsgenie', href: '/opsgenie', icon: Bell, description: 'Mock Opsgenie teams' },
];

function paletteItemClassName() {
  return 'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-muted-foreground cursor-pointer data-[selected=true]:bg-accent data-[selected=true]:text-foreground';
}

function CommandRow({
  value,
  icon: Icon,
  title,
  description,
  onSelect,
}: {
  value: string;
  icon: LucideIcon;
  title: string;
  description?: string;
  onSelect: () => void;
}) {
  return (
    <Command.Item value={value} onSelect={onSelect} className={paletteItemClassName()}>
      <Icon className="h-4 w-4 flex-shrink-0" />
      <div className="min-w-0 flex-1">
        <p className="truncate">{title}</p>
        {description ? <p className="truncate text-xs text-muted-foreground">{description}</p> : null}
      </div>
      <ArrowRight className="h-3.5 w-3.5 opacity-0 data-[selected=true]:opacity-100" />
    </Command.Item>
  );
}

export function CommandPalette() {
  const router = useRouter();
  const { commandPaletteOpen, setCommandPaletteOpen } = useUIStore();
  const [search, setSearch] = useState('');

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setCommandPaletteOpen(!commandPaletteOpen);
      }
      if (e.key === 'Escape') {
        setCommandPaletteOpen(false);
      }
    }
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [commandPaletteOpen, setCommandPaletteOpen]);

  const navigate = useCallback(
    (href: string) => {
      router.push(href);
      setCommandPaletteOpen(false);
      setSearch('');
    },
    [router, setCommandPaletteOpen],
  );

  if (!commandPaletteOpen) return null;

  return (
    <OverlayShell onClose={() => setCommandPaletteOpen(false)}>
      <div className="fixed top-[20%] left-1/2 -translate-x-1/2 w-full max-w-lg">
        <Command className="rounded-xl border border-border bg-popover shadow-2xl overflow-hidden" shouldFilter>
          <div className="flex items-center border-b border-border px-4">
            <Search className="h-4 w-4 text-muted-foreground flex-shrink-0" />
            <Command.Input
              value={search}
              onValueChange={setSearch}
              placeholder="Search pages..."
              className="flex-1 h-12 px-3 bg-transparent text-sm text-foreground placeholder:text-muted-foreground
                focus:outline-none"
            />
          </div>
          <Command.List className="max-h-80 overflow-y-auto p-2">
            <Command.Empty className="px-3 py-8 text-center text-sm text-muted-foreground">
              No matching pages
            </Command.Empty>
            <Command.Group heading="Pages" className="text-xs font-medium text-muted-foreground px-2 py-1.5">
              {pages.map((page) => (
                <CommandRow
                  key={page.href}
                  value={page.name}
                  icon={page.icon}
                  title={page.name}
                  description={page.description}
                  onSelect={() => navigate(page.href)}
                />
              ))}
            </Command.Group>
          </Command.List>
        </Command>
      </div>
    </OverlayShell>
  );
}
