import { createFileRoute, useParams } from '@tanstack/react-router';
import { MockWorkspace } from '@/components/mock/workspace';

export const Route = createFileRoute('/googlechat')({
  component: GoogleChatPage,
});

function GoogleChatPage() {
  const { channelId } = useParams({ strict: false });
  return <MockWorkspace provider="googlechat" channelId={channelId} />;
}
