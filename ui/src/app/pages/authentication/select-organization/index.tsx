import { useState, type FormEvent } from 'react';
import { ArrowRight } from '@carbon/icons-react';
import { Link } from '@carbon/react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { ComboBox, PrimaryButton, Text } from '@/app/components/ui/primitives';

export interface OrganizationOption {
  id: string;
  name: string;
}

interface SelectOrganizationPageProps {
  organizations: OrganizationOption[];
  onContinue: (organization: OrganizationOption) => void;
  onSignOut: () => void;
}

export function SelectOrganizationPage({
  organizations,
  onContinue,
  onSignOut,
}: SelectOrganizationPageProps) {
  const [selectedOrganization, setSelectedOrganization] =
    useState<OrganizationOption | null>(null);

  const continueWithOrganization = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (selectedOrganization) onContinue(selectedOrganization);
  };

  return (
    <div
      className="flex w-full max-w-[37rem] flex-col bg-surface"
      data-testid="organization-selection-panel"
    >
      <Helmet title="Select an organization" />
      <header className="p-6 sm:p-8">
        <Text as="h1" variant="heading-05" className="m-0">
          Select an organization
        </Text>
        <Text as="p" variant="body-02" className="mt-2 text-muted">
          Choose the organization you want to work in. You can switch
          organizations later.
        </Text>
      </header>

      <form noValidate className="p-0" onSubmit={continueWithOrganization}>
        <div className="mx-6 flex flex-col sm:mx-8">
          <ComboBox
            id="organization-selection"
            titleText="Organization"
            placeholder="Search organizations"
            items={organizations}
            itemToString={organization =>
              organization ? `${organization.name} (${organization.id})` : ''
            }
            selectedItem={selectedOrganization}
            onChange={({ selectedItem }) =>
              setSelectedOrganization(selectedItem)
            }
            size="lg"
            disabled={organizations.length === 0}
          />
        </div>

        <div
          className="p-6 sm:p-8"
          data-testid="organization-selection-actions"
        >
          <div className="flex justify-end">
            <Link as="button" inline type="button" onClick={onSignOut}>
              <Text variant="body-compact-01">Sign out</Text>
            </Link>
          </div>
          <PrimaryButton
            size="lg"
            type="submit"
            renderIcon={ArrowRight}
            className="mt-6! w-full! max-w-none! justify-between!"
            disabled={!selectedOrganization}
          >
            Continue
          </PrimaryButton>
        </div>
      </form>
    </div>
  );
}
