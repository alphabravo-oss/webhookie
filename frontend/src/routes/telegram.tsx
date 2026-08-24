import { createFileRoute, useParams } from '@tanstack/react-router';
import { MockWorkspace } from '@/components/mock/workspace';

export const Route = createFileRoute('/telegram')({
  component: TelegramPage,
});

function TelegramPage() {
  const { channelId } = useParams({ strict: false });
  return <MockWorkspace provider="telegram" channelId={channelId} />;
}
