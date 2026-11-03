import { IconOnlyButton, ModalHeader } from '@/app/components/ui/primitives';
import type { ModalProps } from '@/app/components/ui/primitives';
import type { HTMLAttributes, MouseEvent, ReactNode } from 'react';
import useMeasure from 'react-use-measure';
import {
  useDragControls,
  useMotionValue,
  useAnimate,
  motion,
} from 'framer-motion';
import { cn } from '@/utils';
import { Close } from '@carbon/icons-react';

export interface SideModalProps
  extends ModalProps,
    HTMLAttributes<HTMLDivElement> {
  title?: string;
  label?: string;
  children: ReactNode;
  loading?: boolean;
  resizable?: boolean;
}

export function RightSideModal({
  title,
  label,
  modalOpen,
  setModalOpen,
  children,
  className,
  resizable = true,
  onClick,
  ...attributes
}: SideModalProps) {
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

  const handleDrawerClick = (event: MouseEvent<HTMLDivElement>) => {
    event.stopPropagation();
    onClick?.(event);
  };

  return (
    <>
      {modalOpen && (
        <motion.div
          ref={scope}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          onClick={handleClose}
          className="fixed inset-0 z-50 [background:var(--cds-overlay)]"
        >
          <motion.div
            {...attributes}
            role="dialog"
            aria-modal="true"
            aria-label={title || label || 'Details'}
            id="drawer"
            ref={drawerRef}
            onClick={handleDrawerClick}
            initial={{ x: '100%' }}
            animate={{ x: '0%' }}
            transition={{
              ease: 'easeInOut',
            }}
            className={cn(
              'absolute right-0 top-0 h-full min-w-80 overflow-hidden bg-layer text-foreground shadow-lg',
              className,
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
            {resizable && (
              <div className="absolute left-0 bottom-0 top-0 z-10 flex justify-center">
                <button
                  type="button"
                  aria-label="Resize drawer"
                  onPointerDown={e => {
                    controls.start(e);
                  }}
                  className="h-1/2 my-auto w-2 cursor-grab touch-none rounded-[2px] bg-border-strong hover:bg-primary active:cursor-grabbing"
                />
              </div>
            )}
            <div className="relative z-0 h-full overflow-auto flex flex-col">
              {title || label ? (
                <ModalHeader
                  title={title}
                  label={label}
                  onClose={handleClose}
                />
              ) : (
                <header className="absolute top-0 z-10 right-0 p-4">
                  <IconOnlyButton
                    kind="ghost"
                    size="sm"
                    iconDescription="Close"
                    renderIcon={Close}
                    onClick={handleClose}
                  />
                </header>
              )}
              {children}
            </div>
          </motion.div>
        </motion.div>
      )}
    </>
  );
}
