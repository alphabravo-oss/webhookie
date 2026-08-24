import { createFileRoute, useParams } from '@tanstack/react-router';
import { MockWorkspace } from '@/components/mock/workspace';

export const Route = createFileRoute('/mattermost')({
  component: MattermostPage,
});

function MattermostPage() {
  const { channelId } = useParams({ strict: false });
  return <MockWorkspace provider="mattermost" channelId={channelId} />;
}
