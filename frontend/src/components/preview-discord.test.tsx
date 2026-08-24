import { render, screen } from '@testing-library/react';
import { DiscordPreview } from './preview-discord';

describe('DiscordPreview', () => {
  it('disables components after an interaction without rewriting labels', () => {
    render(
      <DiscordPreview
        taken={{ actionId: 'approve', text: 'Approve' }}
        onAction={vi.fn()}
        body={JSON.stringify({
          content: 'Ship v1.4.2?',
          components: [{ components: [{ type: 2, style: 3, label: 'Approve', custom_id: 'approve' }] }],
        })}
      />,
    );
    const btn = screen.getByRole('button', { name: 'Approve' });
    expect(btn).toBeDisabled();
    expect(btn).not.toHaveTextContent('✓');
    expect(screen.getByText('webhookie used Approve')).toBeInTheDocument();
  });
});
