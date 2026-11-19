import { useMemo, useState } from 'react';
import { ChevronDown } from '@carbon/icons-react';
import {
  Popover,
  PopoverContent,
  Search,
  TreeNode,
  TreeView,
} from '@carbon/react';
import type {
  AuthenticationOrganization,
  AuthenticationProject,
} from '@rapidaai/react';
import { GhostButton, Text } from '@/app/components/ui/primitives';

interface OrganizationProjectSwitcherProps {
  organizations: AuthenticationOrganization.AsObject[];
  currentOrganization?: AuthenticationOrganization.AsObject;
  currentProject?: AuthenticationProject.AsObject;
  isLoading?: boolean;
  onSelect: (
    organization: AuthenticationOrganization.AsObject,
    project?: AuthenticationProject.AsObject,
  ) => void;
}

export function OrganizationProjectSwitcher({
  organizations,
  currentOrganization,
  currentProject,
  isLoading = false,
  onSelect,
}: OrganizationProjectSwitcherProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const selectedNodeIDs = currentProject
    ? [`project-${currentOrganization?.id}-${currentProject.id}`]
    : currentOrganization
      ? [`organization-${currentOrganization.id}`]
      : [];
  const currentContext = currentOrganization
    ? `${currentOrganization.name}${currentProject ? ` / ${currentProject.name}` : ''}`
    : 'No organization selected';
  const visibleOrganizations = useMemo(() => {
    const searchTerm = search.trim().toLocaleLowerCase();
    if (!searchTerm) return organizations;

    return organizations.flatMap(organization => {
      if (organization.name.toLocaleLowerCase().includes(searchTerm))
        return [organization];
      const projectsList = organization.projectsList.filter(project =>
        project.name.toLocaleLowerCase().includes(searchTerm),
      );
      return projectsList.length > 0 ? [{ ...organization, projectsList }] : [];
    });
  }, [organizations, search]);

  return (
    <Popover
      align="bottom-end"
      caret={false}
      open={isOpen}
      onRequestClose={() => {
        setIsOpen(false);
        setSearch('');
      }}
    >
      <GhostButton
        aria-expanded={isOpen}
        aria-haspopup="tree"
        aria-label={`Change organization or project. Current context: ${currentContext}`}
        className="h-12! w-64! max-w-64! justify-between! max-sm:min-w-12! max-sm:w-12! max-sm:px-4!"
        disabled={organizations.length === 0}
        isLoading={isLoading}
        renderIcon={ChevronDown}
        size="sm"
        onClick={() => {
          setSearch('');
          setIsOpen(open => !open);
        }}
      >
        <span className="min-w-0 truncate max-sm:sr-only">
          {currentContext}
        </span>
      </GhostButton>
      {isOpen && (
        <PopoverContent className="w-80 p-0">
          <Search
            autoFocus
            closeButtonLabelText="Clear search"
            labelText="Search organizations or projects"
            placeholder="Search organizations or projects"
            size="lg"
            value={search}
            onChange={event => setSearch(event.target.value)}
          />
          <div className="max-h-96 overflow-y-auto py-2">
            {visibleOrganizations.length > 0 ? (
              <TreeView
                hideLabel
                label="Organizations and projects"
                selected={selectedNodeIDs}
                size="sm"
                onSelect={(_event, node) => {
                  const [kind, organizationID, projectID] = String(
                    node.value || '',
                  ).split(':');
                  const organization = organizations.find(
                    candidate => candidate.id === organizationID,
                  );
                  if (!organization) return;
                  if (kind === 'project') {
                    const project = organization.projectsList.find(
                      candidate => candidate.id === projectID,
                    );
                    if (!project) return;
                    onSelect(organization, project);
                  } else if (kind === 'organization') {
                    onSelect(organization);
                  } else {
                    return;
                  }
                  setIsOpen(false);
                  setSearch('');
                }}
              >
                {visibleOrganizations.map(organization => (
                  <TreeNode
                    key={organization.id}
                    id={`organization-${organization.id}`}
                    isExpanded
                    label={organization.name}
                    selected={selectedNodeIDs}
                    value={`organization:${organization.id}`}
                  >
                    {organization.projectsList.length > 0
                      ? organization.projectsList.map(project => (
                          <TreeNode
                            key={project.id}
                            id={`project-${organization.id}-${project.id}`}
                            label={project.name}
                            selected={selectedNodeIDs}
                            value={`project:${organization.id}:${project.id}`}
                          />
                        ))
                      : undefined}
                  </TreeNode>
                ))}
              </TreeView>
            ) : (
              <Text as="p" variant="body-compact-01" className="px-4 py-3">
                No organizations or projects found.
              </Text>
            )}
          </div>
        </PopoverContent>
      )}
    </Popover>
  );
}
