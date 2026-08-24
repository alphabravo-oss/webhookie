import { createFileRoute, useParams } from '@tanstack/react-router';
import { MockWorkspace } from '@/components/mock/workspace';

export const Route = createFileRoute('/discord')({
  component: DiscordPage,
});

function DiscordPage() {
  const { channelId } = useParams({ strict: false });
  return <MockWorkspace provider="discord" channelId={channelId} />;
}
