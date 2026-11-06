import { cn } from '@/utils';
import { Checkmark } from '@carbon/icons-react';
import type { HTMLAttributes, ReactElement } from 'react';
import { Notification } from '@/app/components/ui/feedback/notification';

interface TabFormProps extends HTMLAttributes<HTMLDivElement> {
  activeTab?: string;
  onChangeActiveTab: (code: string) => void;
  errorMessage?: string;
  formHeading?: string;
  form: {
    code: string;
    name: string;
    description?: string;
    body: ReactElement;
    actions: ReactElement[];
  }[];
}

export function TabForm({
  activeTab,
  onChangeActiveTab,
  errorMessage,
  formHeading,
  form,
}: TabFormProps) {
  const activeIndex = form.findIndex(f => f.code === activeTab);

  return (
    <section className="flex flex-1 min-h-0 overflow-hidden">
      <aside className="w-80 hidden md:flex flex-col shrink-0 border-r border-border-subtle bg-layer ">
        <div className="px-6 pt-6 pb-5 border-b border-border-subtle ">
          <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-muted mb-1.5">
            Setup Progress
          </p>
          {formHeading && (
            <p className="text-xs text-muted leading-relaxed">{formHeading}</p>
          )}
        </div>

        <nav className="flex-1 px-6 py-6 overflow-y-auto">
          <ol>
            {form.map((item, index) => {
              const isActive = item.code === activeTab;
              const isCompleted = index < activeIndex;
              const isLast = index === form.length - 1;

              return (
                <li key={item.code} className="relative flex gap-0">
                  <div className="flex flex-col items-center mr-4 shrink-0">
                    <span
                      className={cn(
                        'flex-shrink-0 flex items-center justify-center w-8 h-8 rounded-full text-xs font-semibold transition-colors duration-150',
                        isCompleted && 'bg-primary text-on-primary',
                        isActive &&
                          !isCompleted &&
                          'bg-primary text-on-primary ring-4 ring-primary/10',
                        !isActive &&
                          !isCompleted &&
                          'bg-surface border-2 border-border-subtle text-muted',
                      )}
                    >
                      {isCompleted ? (
                        <Checkmark className="w-3.5 h-3.5" />
                      ) : (
                        <span>{String(index + 1).padStart(2, '0')}</span>
                      )}
                    </span>
                    {!isLast && (
                      <span
                        className={cn(
                          'w-px flex-1 min-h-8',
                          isCompleted ? 'bg-primary' : 'bg-layer ',
                        )}
                      />
                    )}
                  </div>

                  <button
                    type="button"
                    className={cn(
                      'flex flex-col pb-8 pt-0.5 text-left min-w-0 flex-1',
                      isLast && 'pb-0',
                    )}
                    onClick={() => onChangeActiveTab(item.code)}
                  >
                    <span
                      className={cn(
                        'text-[10px] font-semibold tracking-[0.1em] uppercase mb-0.5',
                        isActive ? 'text-primary' : 'text-muted',
                      )}
                    >
                      Step {index + 1}
                    </span>
                    <span
                      className={cn(
                        'text-sm font-medium leading-snug',
                        isActive
                          ? 'text-foreground '
                          : isCompleted
                            ? 'text-muted'
                            : 'text-muted',
                      )}
                    >
                      {item.name}
                    </span>
                    {item.description && (
                      <span className="text-xs text-muted mt-1 leading-relaxed">
                        {item.description}
                      </span>
                    )}
                  </button>
                </li>
              );
            })}
          </ol>
        </nav>
      </aside>

      {/* Content area: flex column so footer sits below scroll */}
      <div className="flex-1 min-h-0 flex flex-col bg-surface ">
        <div className="flex-1 min-h-0 overflow-y-auto flex flex-col">
          {form.map(
            item =>
              item.code === activeTab && (
                <div
                  key={`form-body-${item.code}`}
                  className="flex flex-col flex-1"
                >
                  <header className="px-4 pt-8 pb-6 border-b border-border-subtle ">
                    <p className="text-[10px] font-semibold tracking-[0.12em] uppercase text-muted mb-1.5">
                      Step {activeIndex + 1} of {form.length}
                    </p>
                    <h1 className="text-xl font-semibold text-foreground leading-tight">
                      {item.name}
                    </h1>
                    {item.description && (
                      <p className="text-sm text-muted mt-1.5 leading-relaxed">
                        {item.description}
                      </p>
                    )}
                  </header>

                  {item.body}
                </div>
              ),
          )}
        </div>

        {form.map(
          item =>
            item.code === activeTab && (
              <div key={`footer-${item.code}`} className="shrink-0">
                {errorMessage && (
                  <Notification
                    kind="error"
                    title="Error"
                    subtitle={errorMessage}
                  />
                )}
                {item.actions.map((action, idx) => (
                  <div key={`action-${idx}`}>{action}</div>
                ))}
              </div>
            ),
        )}
      </div>
    </section>
  );
}
