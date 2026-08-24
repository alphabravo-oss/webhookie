import { render, screen } from '@testing-library/react';
import { JsonView, jsonViewPretty } from './json-view';

describe('JsonView', () => {
  it('pretty-prints JSON', () => {
    expect(jsonViewPretty('{"a":1}')).toContain('\n');
    render(<JsonView value='{"a":1}' />);
    expect(screen.getByText(/"a"/)).toBeInTheDocument();
  });
});
