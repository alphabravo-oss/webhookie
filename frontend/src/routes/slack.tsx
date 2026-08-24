import { createFileRoute, useParams } from '@tanstack/react-router';
import { MockWorkspace } from '@/components/mock/workspace';

export const Route = createFileRoute('/slack')({
  component: SlackPage,
});

function SlackPage() {
  const { channelId } = useParams({ strict: false });
  return <MockWorkspace provider="slack" channelId={channelId} />;
}
