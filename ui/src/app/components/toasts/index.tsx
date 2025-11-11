import React, { FC } from 'react';
import toast, { useToaster } from 'react-hot-toast/headless';
import { TickIcon } from '@/app/components/Icon/Tick';
import { cn } from '@/styles/media';
import { CloseIcon } from '@/app/components/Icon/Close';
import { ExclamationTriangleIcon } from '@/app/components/Icon/exclamation-triangle';
import { motion } from 'framer-motion';
import { TriangleAlert } from 'lucide-react';
import {
  GreenNoticeBlock,
  RedNoticeBlock,
} from '@/app/components/container/message/notice-block';

export const Toast = () => {
  const { toasts, handlers } = useToaster();
  const { startPause, endPause, updateHeight } = handlers;

  return (
    <motion.div
      layout
      variants={{
        open: { opacity: 1, x: 0 },
        closed: { opacity: 0, x: '-100%' },
      }}
      animate={{ y: 0, scale: 1 }}
      exit={{ x: '100%', opacity: 0 }}
      transition={{ duration: 0.35, ease: 'easeOut' }}
      onMouseEnter={startPause}
      onMouseLeave={endPause}
      className="sticky top-0 left-0 right-0 w-full z-10"
    >
      {toasts.map(t => {
        const ref = el => {
          if (el && typeof t.height !== 'number') {
            const height = el.getBoundingClientRect().height;
            updateHeight(t.id, height);
          }
        };

        if (t.type === 'success')
          return (
            <div
              ref={ref}
              key={`success_${t.id}`}
              className="bg-white dark:bg-gray-900 rounded-[2px]"
            >
              <SuccessToast
                title={t.message}
                onClose={() => {
                  toast.remove(t.id);
                }}
              />
            </div>
          );

        if (t.type === 'error')
          return (
            <div
              ref={ref}
              key={`success_${t.id}`}
              className="bg-white dark:bg-gray-900 rounded-[2px]"
            >
              <ErrorToast
                title={t.message?.toString()}
                onClose={() => {
                  toast.remove(t.id);
                }}
              />
            </div>
          );
      })}
    </motion.div>
  );
};

export const ErrorToast: FC<{
  title?: any;
  onClose: () => void;
}> = (props: { title?: any; onClose: () => void }) => {
  return (
    <RedNoticeBlock className="flex items-center justify-between text-red-500">
      <div className="flex space-x-2 items-center ">
        <TriangleAlert className="w-4 h-4" strokeWidth={1.5} />
        {props.title && <div className="">{props.title}</div>}
      </div>
      <button className="cursor-pointer" onClick={props.onClose}>
        <CloseIcon className="w-4 h-4" strokeWidth={1.5} />
      </button>
    </RedNoticeBlock>
  );
};

export const SuccessToast: FC<{
  title?: any;
  onClose: () => void;
}> = (props: { title?: any; onClose: () => void }) => {
  return (
    <GreenNoticeBlock className="flex items-center justify-between text-green-500">
      <div className="flex space-x-2 items-center">
        <TickIcon className="w-4 h-4" />
        {props.title && <div className="">{props.title}</div>}
      </div>
      <button className="cursor-pointer" onClick={props.onClose}>
        <CloseIcon className="w-4 h-4" strokeWidth={1.5} />
      </button>
    </GreenNoticeBlock>
  );
};
