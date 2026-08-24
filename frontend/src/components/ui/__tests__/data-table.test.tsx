import { fireEvent, render, screen } from '@testing-library/react';
import { DataTable, type Column } from '@/components/ui/data-table';

type Row = {
  id: string;
  name: string;
  status: string;
};

const rows: Row[] = [
  { id: 'one', name: 'One', status: 'ready' },
  { id: 'two', name: 'Two', status: 'pending' },
];

const columns: Column<Row>[] = [
  {
    key: 'name',
    header: 'Name',
    accessor: (row) => row.name,
  },
  {
    key: 'status',
    header: 'Status',
    accessor: (row) => row.status,
  },
];

describe('DataTable', () => {
  it('renders bulk actions with selected rows', () => {
    render(
      <DataTable
        data={rows}
        columns={columns}
        keyExtractor={(row) => row.id}
        selectable
        bulkActions={(selected) => <button>Delete {selected.length}</button>}
      />
    );

    const [, firstRowCheckbox] = screen.getAllByRole('checkbox');
    fireEvent.click(firstRowCheckbox);

    expect(screen.getByText('1 row selected')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Delete 1' })).toBeInTheDocument();
  });

  it('reserves the requested number of loading rows', () => {
    render(
      <DataTable
        data={[]}
        columns={columns}
        keyExtractor={(row) => row.id}
        loading
        loadingRows={3}
        density="compact"
      />
    );

    expect(screen.getByRole('columnheader', { name: /name/i })).toHaveClass('py-2');
    expect(screen.getAllByRole('row')).toHaveLength(4);
  });
});
