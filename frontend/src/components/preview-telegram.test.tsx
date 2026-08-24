import { fireEvent, render, screen } from '@testing-library/react';
import { TelegramPreview } from './preview-telegram';

describe('TelegramPreview', () => {
  it('fires callback_data on inline keyboard click', () => {
    const onAction = vi.fn();
    render(
      <TelegramPreview
        onAction={onAction}
        body={JSON.stringify({
          chat_id: 1,
          text: 'Ship it?',
          reply_markup: { inline_keyboard: [[{ text: 'Approve', callback_data: 'ok' }]] },
        })}
      />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Approve' }));
    expect(onAction).toHaveBeenCalledWith({ actionId: 'ok', value: 'ok', text: 'Approve' });
  });

  it('removes the inline keyboard after a callback', () => {
    render(
      <TelegramPreview
        taken={{ actionId: 'ok', text: 'Approve' }}
        body={JSON.stringify({
          chat_id: 1,
          text: 'Ship it?',
          reply_markup: { inline_keyboard: [[{ text: 'Approve', callback_data: 'ok' }]] },
        })}
      />,
    );
    expect(screen.queryByRole('button', { name: 'Approve' })).not.toBeInTheDocument();
    expect(screen.getByText('You pressed Approve')).toBeInTheDocument();
  });
});
