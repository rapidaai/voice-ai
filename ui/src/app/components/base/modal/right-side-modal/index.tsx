import { ModalProps } from '@/app/components/base/modal';
import { HTMLAttributes, FC } from 'react';
import useMeasure from 'react-use-measure';
import {
  useDragControls,
  useMotionValue,
  useAnimate,
  motion,
} from 'framer-motion';
import { CloseIcon } from '@/app/components/Icon/Close';
import { TitleHeading } from '@/app/components/Heading/TitleHeading';
import { cn } from '@/styles/media';
/**
 *
 */
export interface SideModalProps
  extends ModalProps,
    HTMLAttributes<HTMLDivElement> {
  // title
  title?: string;

  // children
  children: any;

  //
  loading?: boolean;
}

// const DragCloseDrawer = ({ open, setModalOpen, children }) => {
export const RightSideModal: FC<SideModalProps> = ({
  title,
  modalOpen,
  setModalOpen,
  children,
  className,
}) => {
  const [scope, animate] = useAnimate();
  const [drawerRef, { width }] = useMeasure();

  const x = useMotionValue(0);
  const controls = useDragControls();

  const handleClose = async () => {
    animate(scope.current, {
      opacity: [1, 0],
    });
    const xStart = typeof x.get() === 'number' ? x.get() : 0;
    await animate('#drawer', {
      x: [xStart, width],
    });

    setModalOpen(false);
  };

  return (
    <>
      {modalOpen && (
        <motion.div
          ref={scope}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          onClick={handleClose}
          className="fixed inset-0 z-50 bg-neutral-950/70"
        >
          <motion.div
            id="drawer"
            ref={drawerRef}
            onClick={e => e.stopPropagation()}
            initial={{ x: '100%' }}
            animate={{ x: '0%' }}
            transition={{
              ease: 'easeInOut',
            }}
            className={cn(
              className,
              'absolute right-0 top-0 h-full min-w-[45vw] overflow-hidden bg-white dark:bg-gray-900',
            )}
            style={{ x }}
            drag="x"
            dragControls={controls}
            onDragEnd={() => {
              if (x.get() >= 100) {
                handleClose();
              }
            }}
            dragListener={false}
            dragConstraints={{
              left: 0,
              right: 0,
            }}
            dragElastic={{
              left: 0,
              right: 0.5,
            }}
          >
            <div className="absolute left-0 bottom-0 top-0 z-10 flex justify-center">
              <button
                onPointerDown={e => {
                  controls.start(e);
                }}
                className="h-1/2 my-auto w-2 cursor-grab touch-none rounded-[2px] bg-gray-300 dark:bg-slate-700 hover:bg-blue-600 active:cursor-grabbing"
              ></button>
            </div>
            <div className="relative z-0 h-full overflow-auto flex flex-col">
              {title ? (
                <header className="flex justify-between items-center py-4 dark:bg-gray-800 bg-gray-100 px-4 shadow-md sticky top-0 z-10">
                  <TitleHeading className="text-base">{title}</TitleHeading>
                  <span className="cursor-pointer" onClick={handleClose}>
                    <CloseIcon />
                  </span>
                </header>
              ) : (
                <header className="absolute top-0 z-10 right-0 p-4">
                  <span className="cursor-pointer" onClick={handleClose}>
                    <CloseIcon />
                  </span>
                </header>
              )}
              {children}
            </div>
          </motion.div>
        </motion.div>
      )}
    </>
  );
};
