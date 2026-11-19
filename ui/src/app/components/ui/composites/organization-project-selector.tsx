import { useMemo, useState } from 'react';
import {
  ContainedList,
  ContainedListItem,
  Popover,
  PopoverContent,
  Search,
} from '@carbon/react';
import {
  Application,
  Checkmark,
  ChevronDown,
  Enterprise,
} from '@carbon/icons-react';
import { GhostButton, Text } from '@/app/components/ui/primitives';
import { cn } from '@/utils';

export interface OrganizationProjectSelectorProject {
  id: string;
  name: string;
}

export interface OrganizationProjectSelectorOrganization {
  id: string;
  name: string;
  projects: OrganizationProjectSelectorProject[];
}

interface OrganizationProjectSelectorProps {
  organizations: OrganizationProjectSelectorOrganization[];
  selectedOrganizationId?: string;
  selectedProjectId?: string;
  disabled?: boolean;
  isCompact?: boolean;
  isLoading?: boolean;
  className?: string;
  onChange: (
    organization: OrganizationProjectSelectorOrganization,
    project?: OrganizationProjectSelectorProject,
  ) => void;
}

export function OrganizationProjectSelector({
  organizations,
  selectedOrganizationId,
  selectedProjectId,
  disabled = false,
  isCompact = false,
  isLoading = false,
  className,
  onChange,
}: OrganizationProjectSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const selectedOrganization = organizations.find(
    organization => organization.id === selectedOrganizationId,
  );
  const selectedProject = selectedOrganization?.projects.find(
    project => project.id === selectedProjectId,
  );
  const currentOrganizationName = selectedOrganization?.name;
  const currentProjectName = selectedProject?.name;
  const accessibleContext = currentOrganizationName
    ? `Current organization: ${currentOrganizationName}. ${
        currentProjectName
          ? `Current project: ${currentProjectName}.`
          : 'No project selected.'
      }`
    : 'No organization selected.';
  const visibleOrganizations = useMemo(() => {
    const searchTerm = search.trim().toLocaleLowerCase();
    if (!searchTerm) return organizations;

    return organizations.flatMap(organization => {
      if (organization.name.toLocaleLowerCase().includes(searchTerm))
        return [organization];
      const projects = organization.projects.filter(project =>
        project.name.toLocaleLowerCase().includes(searchTerm),
      );
      return projects.length > 0 ? [{ ...organization, projects }] : [];
    });
  }, [organizations, search]);

  return (
    <Popover
      align="bottom-end"
      autoAlign
      caret={false}
      open={isOpen}
      onRequestClose={() => {
        setIsOpen(false);
        setSearch('');
      }}
    >
      <span className={cn('inline-flex h-12', isCompact ? 'w-12' : 'w-72')}>
        <GhostButton
          aria-expanded={isOpen}
          aria-label={`Change organization or project. ${accessibleContext}`}
          className={cn('h-12! w-full! justify-between!', className)}
          disabled={disabled || organizations.length === 0}
          isLoading={isLoading}
          renderIcon={ChevronDown}
          size="sm"
          onClick={() => {
            setSearch('');
            setIsOpen(open => !open);
          }}
        >
          <span
            className={cn('min-w-0 flex-col', isCompact ? 'hidden' : 'flex')}
          >
            <Text as="span" variant="label-01" className="truncate text-muted">
              {currentOrganizationName || 'Organization'}
            </Text>
            <Text as="span" variant="body-compact-01" className="truncate">
              {currentProjectName || 'Select a project'}
            </Text>
          </span>
        </GhostButton>
      </span>

      {isOpen && (
        <PopoverContent className="w-72 p-0">
          <div className="border-b border-border-subtle px-4 py-3">
            <Text as="h2" variant="heading-compact-01">
              Switch context
            </Text>
            <Text as="p" variant="body-compact-01" className="mt-1 text-muted">
              Choose an organization and project.
            </Text>
          </div>
          <Search
            autoFocus
            closeButtonLabelText="Clear search"
            labelText="Search organizations or projects"
            placeholder="Search by name"
            size="lg"
            value={search}
            onChange={event => setSearch(event.target.value)}
          />

          <div className="overflow-y-auto" style={{ maxHeight: '24rem' }}>
            {visibleOrganizations.length > 0 ? (
              visibleOrganizations.map(organization => (
                <ContainedList
                  key={organization.id}
                  isInset
                  kind="disclosed"
                  size="md"
                  label={
                    <span className="-mx-4 flex min-w-0 items-center gap-3 border-y border-border-subtle px-4 py-1.5">
                      <Enterprise className="shrink-0" size={16} />
                      <Text
                        as="span"
                        variant="heading-compact-01"
                        className="truncate text-foreground"
                      >
                        {organization.name}
                      </Text>
                      <Text
                        as="span"
                        variant="label-01"
                        className="ms-auto shrink-0 text-muted"
                      >
                        {organization.projects.length === 1
                          ? '1 project'
                          : `${organization.projects.length} projects`}
                      </Text>
                    </span>
                  }
                >
                  {organization.projects.length > 0 ? (
                    organization.projects.map(project => {
                      const isSelected =
                        organization.id === selectedOrganizationId &&
                        project.id === selectedProjectId;
                      return (
                        <ContainedListItem
                          key={project.id}
                          onClick={() => {
                            onChange(organization, project);
                            setIsOpen(false);
                            setSearch('');
                          }}
                        >
                          <span className="flex min-w-0 items-center gap-3 pl-8">
                            <Application
                              aria-hidden
                              className="shrink-0"
                              size={16}
                            />
                            <Text
                              as="span"
                              variant="body-compact-01"
                              className="truncate"
                            >
                              {project.name}
                            </Text>
                            {isSelected && (
                              <Checkmark
                                aria-label="Selected project"
                                className="ms-auto shrink-0"
                                size={16}
                              />
                            )}
                          </span>
                        </ContainedListItem>
                      );
                    })
                  ) : (
                    <ContainedListItem
                      onClick={() => {
                        onChange(organization, undefined);
                        setIsOpen(false);
                        setSearch('');
                      }}
                    >
                      <span className="flex min-w-0 items-center gap-3 ps-4">
                        <Enterprise
                          aria-hidden
                          className="shrink-0"
                          size={16}
                        />
                        <Text
                          as="span"
                          variant="body-compact-01"
                          className="truncate"
                        >
                          Organization only
                        </Text>
                        <Text
                          as="span"
                          variant="label-01"
                          className="ms-auto shrink-0 text-muted"
                        >
                          Organization
                        </Text>
                      </span>
                    </ContainedListItem>
                  )}
                </ContainedList>
              ))
            ) : (
              <Text
                as="p"
                variant="body-compact-01"
                className="px-4 py-5 text-muted"
              >
                No organizations or projects found.
              </Text>
            )}
          </div>
        </PopoverContent>
      )}
    </Popover>
  );
}
