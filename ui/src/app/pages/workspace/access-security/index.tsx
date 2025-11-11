import React, { useState } from 'react';
import { Switch } from '@headlessui/react';
import { Helmet } from '@/app/components/Helmet';
import { DescriptiveHeading } from '@/app/components/Heading/DescriptiveHeading';
export function AccessSecurityPage() {
  const [enabled, setEnabled] = useState(true);
  return (
    <>
      <Helmet title="Organization Security"></Helmet>

      <div className="space-y-8 my-10">
        <div>
          <DescriptiveHeading heading="Organization Security" />
        </div>
        <section>
          <div className="border rounded-[2px] px-5 py-3 bg-white dark:bg-gray-800 flex dark:border-gray-700">
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
              <a
                href="https://docs.rapida.ai"
                className="text-blue-600 dark:text-blue-400 text-sm flex items-center"
              >
                Read the support documentation
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth="1.5"
                  stroke="currentColor"
                  className="w-4 h-4 ml-1"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M4.5 12h15m0 0l-6.75-6.75M19.5 12l-6.75 6.75"
                  />
                </svg>
              </a>
            </div>
            <div className="">
              <Switch
                disabled={true}
                checked={enabled}
                onChange={setEnabled}
                className={`${enabled ? 'bg-blue-900' : 'bg-blue-700'}
          relative inline-flex h-[25px] w-[40px] shrink-0 cursor-pointer rounded-[2px] border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-hidden focus-visible:ring-2  focus-visible:ring-white/75`}
              >
                <span className="sr-only">Use setting</span>
                <span
                  aria-hidden="true"
                  className={`${enabled ? 'trangray-x-4' : 'trangray-x-0'}
            pointer-events-none inline-block h-[20px] w-[20px] transform rounded-[2px] bg-white shadow-lg ring-0 transition duration-200 ease-in-out`}
                />
              </Switch>
            </div>
          </div>
        </section>
        <section></section>
      </div>
    </>
  );
}
