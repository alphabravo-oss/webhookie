import { createFileRoute, useParams } from '@tanstack/react-router';
import { MockWorkspace } from '@/components/mock/workspace';

export const Route = createFileRoute('/teams')({
  component: TeamsPage,
});

function TeamsPage() {
  const { channelId } = useParams({ strict: false });
  return <MockWorkspace provider="teams" channelId={channelId} />;
}
