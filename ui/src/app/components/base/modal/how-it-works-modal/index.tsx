import { ModalBody } from '@/app/components/base/modal/modal-body';
import React, { FC } from 'react';

export const HowItWorks: FC<{
  steps: Array<{
    title: string;
    icon: React.ReactElement;
    description: string;
  }>;
}> = React.memo(({ steps }) => {
  return (
    <ModalBody>
      <div className="grid grid-flow-col">
        {steps.map((step, index) => (
          <div key={index} className="flex flex-col items-start relative">
            <div className="px-6 opacity-90">
              <div className="bg-blue-100 dark:bg-blue-800/40 p-3 mb-6 w-fit">
                <div className="text-blue-600">{step.icon}</div>
              </div>
              <h3 className="text-base font-medium mb-2">{step.title}</h3>
              <p className="text-sm">{step.description}</p>
            </div>
            {index !== steps.length - 1 && (
              <div className="absolute right-0 w-[2px] top-4 bottom-4 bg-gray-200 dark:bg-gray-800" />
            )}
          </div>
        ))}
      </div>
    </ModalBody>
  );
});
