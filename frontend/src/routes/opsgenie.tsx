import { createFileRoute, useParams } from '@tanstack/react-router';
import { MockWorkspace } from '@/components/mock/workspace';

export const Route = createFileRoute('/opsgenie')({
  component: OpsgeniePage,
});

function OpsgeniePage() {
  const { channelId } = useParams({ strict: false });
  return <MockWorkspace provider="opsgenie" channelId={channelId} />;
}
