import { cn } from '@/utils';
import { FC, HTMLAttributes, ReactElement } from 'react';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';

interface TabFormProps extends HTMLAttributes<HTMLDivElement> {
  activeTab?: string;
  onChangeActiveTab: (code: string) => void; // Added parameter type
  errorMessage?: string;
  formHeading?: string;
  form: {
    code: string;
    name: string;
    description?: string; // Made `description` optional
    body: ReactElement;
    actions: ReactElement[];
  }[];
}

export const TabForm: FC<TabFormProps> = ({
  activeTab,
  onChangeActiveTab,
  errorMessage,
  formHeading,
  form,
}) => {
  return (
    <section className="flex flex-1 max-h-full bg-white dark:bg-gray-900">
      <div className="w-96 hidden md:block p-4">
        <div className="relative p-4">
          <div className="mb-6">
            <h2 className="text-lg font-medium text-foreground mb-1">
              Setup Progress
            </h2>
            <p className="text-sm text-muted-foreground">{formHeading}</p>
          </div>
          <div className="overflow-hidden space-y-8">
            {form.map((item, index) => (
              <div
                className={cn(
                  "relative flex-1 after:content-['']  after:h-full after:inline-block after:absolute after:-bottom-8 after:left-[14px]",
                  index !== form.length - 1 && 'after:w-0.5',
                  'after:bg-blue-600',
                )}
                key={item.code}
                onClick={() => onChangeActiveTab(item.code)}
              >
                <div className={cn('absolute')}>
                  {item.code === activeTab ? (
                    <span className="w-8 h-8 bg-blue-600 border-2 border-transparent rounded-full flex justify-center items-center mr-3 text-sm text-white">
                      <svg
                        className="w-5 h-5 stroke-white"
                        viewBox="0 0 24 24"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          d="M5 12L9.28722 16.2923C9.62045 16.6259 9.78706 16.7927 9.99421 16.7928C10.2014 16.7929 10.3681 16.6262 10.7016 16.2929L20 7"
                          stroke="stroke-current"
                          strokeWidth="1.6"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          className="my-path"
                        ></path>
                      </svg>
                    </span>
                  ) : (
                    <span className="w-8 h-8 bg-white dark:bg-gray-900 border-2 border-blue-600 rounded-full flex justify-center items-center mr-3 text-sm text-blue-600 font-semibold">
                      0{index + 1}
                    </span>
                  )}
                </div>

                {/* Step Information */}
                <div className="ml-16">
                  {item.name && (
                    <>
                      <h4 className="font-medium text-sm">Step {index + 1}</h4>
                      <h4
                        className={cn(
                          'text-base mt-1.5 font-medium',
                          item.code === activeTab && 'text-blue-600',
                        )}
                      >
                        {item.name}
                      </h4>
                    </>
                  )}
                  {item.description && (
                    <span className="block text-sm font-normal mt-1 opacity-85">
                      {item.description}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
      <div className={cn('w-2/3 h-full flex-1 relative')}>
        <div className="h-full flex-1 overflow-auto flex flex-col pb-11">
          {form.map(
            item =>
              item.code === activeTab && (
                <div
                  key={`form-body-${item.code}`}
                  className={cn(
                    'space-y-6 flex-1 flex flex-col border bg-light-background dark:bg-gray-900',
                  )}
                >
                  {item.body}
                  <PageActionButtonBlock errorMessage={errorMessage}>
                    {item.actions.map((action, idx) => (
                      <div key={`action-${idx}`}>{action}</div>
                    ))}
                  </PageActionButtonBlock>
                </div>
              ),
          )}
        </div>
      </div>
    </section>
  );
};
