import { fireEvent, render, screen } from '@testing-library/react';
import { SlackPreview } from './preview-slack';

describe('SlackPreview', () => {
  it('renders section text and a divider', () => {
    render(
      <SlackPreview
        body={JSON.stringify({
          blocks: [
            { type: 'section', text: { type: 'mrkdwn', text: '*deploy failed*' } },
            { type: 'divider' },
          ],
        })}
      />,
    );
    expect(screen.getByText('deploy failed')).toBeInTheDocument();
    expect(document.querySelector('hr')).toBeTruthy();
  });

  it('does not duplicate fallback text when blocks are present', () => {
    render(
      <SlackPreview
        body={JSON.stringify({
          text: 'Deploy to prod?',
          blocks: [{ type: 'section', text: { type: 'plain_text', text: 'Deploy to prod?' } }],
        })}
      />,
    );
    expect(screen.getAllByText('Deploy to prod?')).toHaveLength(1);
  });

  it('fires onAction when a button is clicked', () => {
    const onAction = vi.fn();
    render(
      <SlackPreview
        onAction={onAction}
        body={JSON.stringify({
          blocks: [
            {
              type: 'actions',
              block_id: 'row',
              elements: [{ type: 'button', action_id: 'approve', text: { text: 'Approve' }, value: 'yes' }],
            },
          ],
        })}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Approve' }));
    expect(onAction).toHaveBeenCalledWith({ actionId: 'approve', value: 'yes', text: 'Approve', blockId: 'row' });
  });

  it('marks the clicked button as taken and disables the rest', () => {
    render(
      <SlackPreview
        taken={{ actionId: 'approve', text: 'Approve' }}
        onAction={vi.fn()}
        body={JSON.stringify({
          blocks: [
            {
              type: 'actions',
              elements: [
                { type: 'button', action_id: 'approve', text: { text: 'Approve' }, value: 'yes' },
                { type: 'button', action_id: 'deny', text: { text: 'Deny' }, value: 'no', style: 'danger' },
              ],
            },
          ],
        })}
      />,
    );
    const approve = screen.getByRole('button', { name: '✓ Approve' });
    const deny = screen.getByRole('button', { name: 'Deny' });
    expect(approve).toHaveAttribute('aria-pressed', 'true');
    expect(approve).toBeDisabled();
    expect(deny).toBeDisabled();
    expect(screen.getByText('webhookie clicked Approve')).toBeInTheDocument();
  });
});
