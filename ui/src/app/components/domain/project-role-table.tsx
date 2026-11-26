import { Add, TrashCan } from '@carbon/icons-react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableHeader,
  TableRow,
} from '@carbon/react';
import {
  Dropdown,
  IconOnlyButton,
  MultiSelect,
  TertiaryButton,
} from '@/app/components/ui/primitives';

export type ProjectRoleRow = {
  projectId: string;
  roleDefinitionIds: string[];
};

type ProjectOption = {
  name: string;
  value: string;
};

type RoleOption = {
  name: string;
  value: string;
};

interface ProjectRoleTableProps {
  rows: ProjectRoleRow[];
  onChange: (rows: ProjectRoleRow[]) => void;
  projectOptions: ProjectOption[];
  roleOptions: RoleOption[];
  title?: string;
  addButtonLabel?: string;
  showAddButton?: boolean;
  showRemoveColumn?: boolean;
  defaultProjectId?: string;
}

export function ProjectRoleTable(props: ProjectRoleTableProps) {
  const showAddButton = props.showAddButton ?? true;
  const showRemoveColumn = props.showRemoveColumn ?? true;
  const addButtonLabel = props.addButtonLabel || 'Add project role';
  const columnCount = showRemoveColumn ? 3 : 2;

  return (
    <div>
      <TableContainer
        title={props.title ? `${props.title} (${props.rows.length})` : ''}
        className="![padding-block-start:0] border border-border-subtle"
      >
        <Table className="w-full table-fixed" size="sm" useZebraStyles={false}>
          <TableHead>
            <TableRow>
              <TableHeader>Project</TableHeader>
              <TableHeader>Roles</TableHeader>
              {showRemoveColumn && (
                <TableHeader className="!w-10 !text-center">
                  <span className="sr-only">Actions</span>
                </TableHeader>
              )}
            </TableRow>
          </TableHead>
          <TableBody>
            {props.rows.length === 0 && (
              <TableRow>
                <TableCell colSpan={columnCount}>
                  No project roles yet. Click <strong>{addButtonLabel}</strong>{' '}
                  below.
                </TableCell>
              </TableRow>
            )}
            {props.rows.map((row, index) => (
              <TableRow key={index}>
                <TableCell className="!p-0">
                  <Dropdown
                    autoAlign
                    id={`project-role-project-${index}`}
                    titleText=""
                    label="Select project"
                    items={props.projectOptions}
                    selectedItem={
                      props.projectOptions.find(
                        project => project.value === row.projectId,
                      ) || null
                    }
                    itemToString={(item: ProjectOption | null) =>
                      item?.name || ''
                    }
                    onChange={({ selectedItem }) => {
                      const next = [...props.rows];
                      next[index] = {
                        ...row,
                        projectId: selectedItem?.value || '',
                      };
                      props.onChange(next);
                    }}
                    size="md"
                    className="w-full min-w-0"
                  />
                </TableCell>
                <TableCell className="!p-0">
                  <MultiSelect
                    id={`project-role-role-${index}`}
                    autoAlign
                    titleText="Roles"
                    hideLabel
                    label="Select roles"
                    items={props.roleOptions}
                    clearSelectionDescription="Total items selected: "
                    clearSelectionText="To clear selection, press Delete or Backspace."
                    selectedItems={props.roleOptions.filter(role =>
                      row.roleDefinitionIds.includes(role.value),
                    )}
                    selectionFeedback="top-after-reopen"
                    itemToString={(item: RoleOption) => item?.name || ''}
                    onChange={({ selectedItems }) => {
                      const next = [...props.rows];
                      next[index] = {
                        ...row,
                        roleDefinitionIds: (selectedItems || []).map(
                          role => role.value,
                        ),
                      };
                      props.onChange(next);
                    }}
                    size="md"
                    type="default"
                    className="w-full min-w-0"
                  />
                </TableCell>
                {showRemoveColumn && (
                  <TableCell className="!w-10 !p-0 !text-center">
                    <IconOnlyButton
                      renderIcon={TrashCan}
                      iconDescription="Remove project role"
                      kind="ghost"
                      size="sm"
                      className="!text-support-error"
                      onClick={() =>
                        props.onChange(props.rows.filter((_, i) => i !== index))
                      }
                    />
                  </TableCell>
                )}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      {showAddButton && (
        <div className="mt-4">
          <TertiaryButton
            size="md"
            renderIcon={Add}
            className="!w-full !max-w-none"
            onClick={() =>
              props.onChange([
                ...props.rows,
                {
                  projectId:
                    props.defaultProjectId ||
                    props.projectOptions[0]?.value ||
                    '',
                  roleDefinitionIds: [],
                },
              ])
            }
          >
            {addButtonLabel}
          </TertiaryButton>
        </div>
      )}
    </div>
  );
}
