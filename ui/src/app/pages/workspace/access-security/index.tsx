import React, { useState } from 'react';
import { Link, Toggle } from '@carbon/react';
import { ArrowRight } from '@carbon/icons-react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { DescriptiveHeading } from '@/app/components/layout/heading/descriptive-heading';
import { useDocumentationUrl } from '@/theme/documentation-url';

export function AccessSecurityPage() {
  const [enabled, setEnabled] = useState(true);
  const documentationUrl = useDocumentationUrl();

  return (
    <>
      <Helmet title="Organization Security"></Helmet>

      <div className="space-y-8 my-10">
        <div>
          <DescriptiveHeading heading="Organization Security" />
        </div>
        <section>
          <div className="border rounded-[2px] px-5 py-3 bg-surface flex border-border-subtle">
            <div className="">
              <h3 className="font-medium text-lg ">
                Two-Factor Authentication
              </h3>
              <p className="text-sm my-2">
                Whenever users sign in with a username and password, they also
                need to enter a security code generated on their mobile device.
                Users do not need a security code when signing in through the
                organization's identity provider (SSO).
              </p>
              <Link href={documentationUrl} size="sm" renderIcon={ArrowRight}>
                Read the support documentation
              </Link>
            </div>
            <div className="">
              <Toggle
                id="organization-two-factor-authentication"
                disabled
                toggled={enabled}
                onToggle={setEnabled}
                labelText="Two-factor authentication"
                hideLabel
                size="sm"
              />
            </div>
          </div>
        </section>
        <section></section>
      </div>
    </>
  );
}
