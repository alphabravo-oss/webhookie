import { render, screen } from '@testing-library/react';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';

describe('Table primitives', () => {
  it('renders semantic table markup with default classes', () => {
    render(
      <Table data-testid="inventory-table">
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow>
            <TableCell>astronomer-agent</TableCell>
          </TableRow>
        </TableBody>
      </Table>,
    );

    expect(screen.getByTestId('inventory-table')).toHaveClass('w-full', 'text-sm');
    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: 'astronomer-agent' })).toBeInTheDocument();
  });
});
