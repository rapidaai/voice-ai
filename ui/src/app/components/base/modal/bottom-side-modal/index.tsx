import { ModalProps } from '@/app/components/base/modal';
import { HTMLAttributes, FC } from 'react';
import useMeasure from 'react-use-measure';
import {
  useDragControls,
  useMotionValue,
  useAnimate,
  motion,
} from 'framer-motion';
import { cn } from '@/styles/media';
/**
 *
 */
export interface BottomModalProps
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
export const BottomModal: FC<BottomModalProps> = ({
  title,
  modalOpen,
  setModalOpen,
  children,
  className,
}) => {
  const [scope, animate] = useAnimate();
  const [drawerRef, { width }] = useMeasure();

  const y = useMotionValue(0);
  const controls = useDragControls();

  const handleClose = async () => {
    animate(scope.current, {
      opacity: [1, 0],
    });
    const xStart = typeof y.get() === 'number' ? y.get() : 0;
    await animate('#drawer', {
      y: [xStart, width],
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
            initial={{ y: '100%' }}
            animate={{ y: '0%' }}
            transition={{
              ease: 'easeInOut',
            }}
            className={cn(
              className,
              'absolute left-0 right-0 bottom-0 min-h-[45vw] overflow-hidden bg-white dark:bg-gray-900',
            )}
            style={{ y }}
            drag="y"
            dragControls={controls}
            onDragEnd={() => {
              if (y.get() >= 100) {
                handleClose();
              }
            }}
            dragListener={false}
            dragConstraints={{
              top: 0,
              bottom: 0,
            }}
            dragElastic={{
              top: 0,
              bottom: 0.5,
            }}
          >
            <div className="absolute -top-0 left-0 right-0 z-10 flex justify-center">
              <button
                onPointerDown={e => {
                  controls.start(e);
                }}
                className="w-full my-auto h-2 cursor-grab touch-none rounded-[2px] bg-gray-300 dark:bg-slate-700 hover:bg-blue-600 active:cursor-grabbing"
              ></button>
            </div>
            <div className="relative z-0 h-full overflow-auto flex flex-col">
              {children}
            </div>
          </motion.div>
        </motion.div>
      )}
    </>
  );
};
