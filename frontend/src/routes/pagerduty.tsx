import { createFileRoute, useParams } from '@tanstack/react-router';
import { MockWorkspace } from '@/components/mock/workspace';

export const Route = createFileRoute('/pagerduty')({
  component: PagerDutyPage,
});

function PagerDutyPage() {
  const { channelId } = useParams({ strict: false });
  return <MockWorkspace provider="pagerduty" channelId={channelId} />;
}
