import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/mattermost/$channelId')({
  component: () => null,
});
