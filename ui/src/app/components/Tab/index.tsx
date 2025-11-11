import { TabBody } from '@/app/components/Tab/TabBody';
import { TabHeader } from '@/app/components/Tab/TabHeader';
import { cn } from '@/styles/media';
import React, { FC, HTMLAttributes, useState } from 'react';

export interface TabProps extends HTMLAttributes<HTMLDivElement> {
  active: string;
  tabs: { label: string; element: React.ReactElement }[];
  strict?: boolean;
  linkClass?: string;
}
export const Tab: FC<TabProps> = ({
  active,
  tabs,
  className,
  strict = true,
}) => {
  const [isActive, setIsActive] = useState(active);
  return (
    <>
      <TabHeader className={className}>
        <div className="flex items-center divide-x border-r w-fit">
          {tabs.map((ix, id) => {
            return (
              <div
                key={id}
                onClick={() => {
                  setIsActive(ix.label);
                }}
                className={cn(
                  'group cursor-pointer hover:bg-gray-500/10',
                  isActive === ix.label ? 'bg-gray-500/10 text-blue-500' : '',
                )}
              >
                <div className="px-6 py-2 font-semibold whitespace-nowrap tracking-wide text-pretty capitalize">
                  {ix.label}
                </div>
              </div>
            );
          })}
        </div>
      </TabHeader>
      {strict
        ? tabs.map((ix, id) => {
            return (
              <TabBody
                key={id}
                className={cn(ix.label === isActive ? 'flex' : 'hidden')}
              >
                {ix.element}
              </TabBody>
            );
          })
        : tabs
            .filter(x => x.label === isActive)
            .map((ix, id) => {
              return <TabBody key={id}>{ix.element}</TabBody>;
            })}
    </>
  );
};

export const SideTab: FC<TabProps> = ({
  active,
  tabs,
  className,
  strict = true,
}) => {
  const [isActive, setIsActive] = useState(active);
  return (
    <>
      <TabHeader className={cn(className, 'border-none')}>
        <div className="flex flex-col border-r h-full">
          {tabs.map((ix, id) => {
            return (
              <div
                key={id}
                onClick={() => {
                  setIsActive(ix.label);
                }}
                className={cn(
                  'group px-2 border-r-[3px] border-transparent -ms-[0.1rem] cursor-pointer',
                  isActive === ix.label
                    ? 'border-blue-500! text-blue-500 bg-blue-500/5'
                    : '',
                )}
              >
                <div className="capitalize px-3 py-3 font-medium 2xl:text-base">
                  {ix.label}
                </div>
              </div>
            );
          })}
        </div>
      </TabHeader>
      {strict
        ? tabs.map((ix, id) => {
            return (
              <TabBody
                key={id}
                className={cn(ix.label === isActive ? 'block' : 'hidden')}
              >
                {ix.element}
              </TabBody>
            );
          })
        : tabs
            .filter(x => x.label === isActive)
            .map((ix, id) => {
              return <TabBody key={id}>{ix.element}</TabBody>;
            })}
    </>
  );
};
