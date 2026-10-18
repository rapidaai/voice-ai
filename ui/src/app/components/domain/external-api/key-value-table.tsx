import { Add, TrashCan } from '@carbon/icons-react';
import {
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
} from '@carbon/react';
import { TextInput } from '@/app/components/ui/primitives/form';
import { tableFormContainerClassName } from '@/app/components/ui/table';

export interface KeyValueRow {
  key: string;
  value: string;
}

interface KeyValueTableProps {
  addButtonLabel: string;
  emptyLabel: string;
  keyInputPrefix: string;
  rows: KeyValueRow[];
  onChange: (rows: KeyValueRow[]) => void;
  valueInputPrefix: string;
}

export function KeyValueTable({
  addButtonLabel,
  emptyLabel,
  keyInputPrefix,
  rows,
  onChange,
  valueInputPrefix,
}: KeyValueTableProps) {
  const updateRow = (
    index: number,
    field: keyof KeyValueRow,
    value: string,
  ) => {
    onChange(
      rows.map((row, rowIndex) =>
        rowIndex === index ? { ...row, [field]: value } : row,
      ),
    );
  };

  return (
    <>
      <TableContainer className={tableFormContainerClassName}>
        <Table size="sm" useZebraStyles={false}>
          <TableHead>
            <TableRow>
              <TableHeader>Key</TableHeader>
              <TableHeader>Value</TableHeader>
              <TableHeader className="!w-10 !text-center">Actions</TableHeader>
            </TableRow>
          </TableHead>
          <TableBody>
            {rows.length === 0 && (
              <TableRow>
                <TableCell colSpan={3}>
                  No {emptyLabel} yet. Click <strong>{addButtonLabel}</strong>{' '}
                  below.
                </TableCell>
              </TableRow>
            )}
            {rows.map((row, index) => (
              <TableRow key={`${row.key}-${index}`}>
                <TableCell className="!p-0">
                  <TextInput
                    id={`${keyInputPrefix}-${index}`}
                    data-testid={`${keyInputPrefix}-${index}`}
                    labelText="Key"
                    hideLabel
                    value={row.key}
                    onChange={event =>
                      updateRow(index, 'key', event.target.value)
                    }
                    placeholder="Key"
                    size="md"
                  />
                </TableCell>
                <TableCell className="!p-0">
                  <TextInput
                    id={`${valueInputPrefix}-${index}`}
                    data-testid={`${valueInputPrefix}-${index}`}
                    labelText="Value"
                    hideLabel
                    value={row.value}
                    onChange={event =>
                      updateRow(index, 'value', event.target.value)
                    }
                    placeholder="Value"
                    size="md"
                  />
                </TableCell>
                <TableCell className="!w-10 !p-0 !text-center">
                  <Button
                    hasIconOnly
                    renderIcon={TrashCan}
                    iconDescription="Remove"
                    kind="ghost"
                    size="sm"
                    className="!text-[var(--cds-support-error)]"
                    onClick={() => onChange(rows.filter((_, i) => i !== index))}
                  />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      <div className="pt-2">
        <Button
          kind="tertiary"
          size="md"
          renderIcon={Add}
          onClick={() => onChange([...rows, { key: '', value: '' }])}
          className="!w-full !max-w-none"
        >
          {addButtonLabel}
        </Button>
      </div>
    </>
  );
}
