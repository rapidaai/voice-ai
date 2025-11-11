import React from 'react';
import { Disclosure as DS, Transition } from '@headlessui/react';

interface DisclosureProps {
  open: boolean;
  children?: any;
}
export function Disclosure(props: DisclosureProps) {
  return (
    <DS defaultOpen={props.open}>
      {({ open }) => (
        <>
          <Transition
            show={props.open}
            enter="transition duration-100 origin-top"
            enterFrom="transform scale-y-0"
            enterTo="transform scale-y-100 opacity-100"
            leave="transition duration-100 origin-top"
            leaveFrom="transform scale-y-100 opacity-100"
            leaveTo="transform scale-y-0 opacity-0"
          >
            <DS.Panel static>{props.children}</DS.Panel>
          </Transition>
        </>
      )}
    </DS>
  );
}
