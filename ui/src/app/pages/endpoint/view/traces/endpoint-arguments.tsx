import { Argument, Metadata } from '@rapidaai/react';
import { Table } from '@/app/components/base/tables/table';
import { TableBody } from '@/app/components/base/tables/table-body';
import { TableCell } from '@/app/components/base/tables/table-cell';
import { TableHead } from '@/app/components/base/tables/table-head';
import { TableRow } from '@/app/components/base/tables/table-row';
import { BlueNoticeBlock } from '@/app/components/container/message/notice-block';
import { FC } from 'react';

export const EndpointArguments: FC<{ args: Array<Argument> }> = ({ args }) => {
  if (args.length <= 0)
    return (
      <BlueNoticeBlock>
        There are no args for given endpoint execution.
      </BlueNoticeBlock>
    );
  return (
    <Table className="w-full">
      <TableHead
        columns={[
          { name: 'Name', key: 'Name' },
          { name: 'Value', key: 'Value' },
        ]}
      />
      <TableBody>
        {args.map((ar, index) => {
          return (
            <TableRow key={index}>
              <TableCell>{ar.getName()}</TableCell>
              <TableCell className="break-words break-all">
                {ar.getValue()}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
};
