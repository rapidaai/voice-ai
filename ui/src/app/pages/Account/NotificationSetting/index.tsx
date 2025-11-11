import { useState } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { Switch } from '@headlessui/react';
export function NotificationSettingPage() {
  const [enabled, setEnabled] = useState(false);
  return (
    <>
      <Helmet title="Notification Settings" />
      <div className="space-y-8 my-10">
        <section>
          <ul>
            <li className="flex items-center border-gray-200 dark:border-gray-700 justify-between border-b pb-3 pt-1">
              {/* Left */}
              <div>
                <div className="text-gray-800 dark:text-gray-100 font-medium py-1">
                  Activities
                </div>
                <div className="text-sm">
                  Running status of test suite executions, when completed,
                  failed or paused.
                </div>
              </div>
              {/* Right */}
              <div className="flex items-center ml-4">
                <div
                  className="text-sm mr-2 ciz4v czgoy clmtf"
                  x-text="checked ? 'On' : 'Off'"
                >
                  On
                </div>
                <Switch
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
            </li>
            <li className="flex items-center border-gray-200 dark:border-gray-700 justify-between border-b  pb-3 pt-1">
              {/* Left */}
              <div>
                <div className="text-gray-800 dark:text-gray-100 font-medium py-1">
                  Deployments
                </div>
                <div className="text-sm">
                  Change in prompt version or endpoint version change will be
                  notified by email.
                </div>
              </div>
              {/* Right */}
              <div className="flex items-center ml-4">
                <div
                  className="text-sm mr-2 ciz4v czgoy clmtf"
                  x-text="checked ? 'On' : 'Off'"
                >
                  On
                </div>
                <Switch
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
            </li>
          </ul>
        </section>
        {/* Shares */}
        <section>
          <ul>
            <li className="flex items-center border-gray-200 dark:border-gray-700 justify-between border-b  pb-3 pt-1">
              {/* Left */}
              <div>
                <div className="text-gray-800 dark:text-gray-100 font-medium py-1">
                  Provider Apis
                </div>
                <div className="text-sm">
                  Failure with your provider APis like rate limit from openAI,
                  or invalid token response from any providers.
                </div>
              </div>
              {/* Right */}
              <div className="flex items-center ml-4">
                <div
                  className="text-sm mr-2 ciz4v czgoy clmtf"
                  x-text="checked ? 'On' : 'Off'"
                >
                  On
                </div>
                <Switch
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
            </li>
          </ul>
        </section>
      </div>
    </>
  );
}
