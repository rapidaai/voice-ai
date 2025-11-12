import { VaultCredential } from '@rapidaai/react';
import { Dropdown } from '@/app/components/dropdown';
import { FormLabel } from '@/app/components/form-label';
import { IButton } from '@/app/components/form/button';
import { FieldSet } from '@/app/components/form/fieldset';
import { cn } from '@/utils';
import { Plus, RotateCcw } from 'lucide-react';
import { FC, useCallback, useEffect, useState } from 'react';
import { CreateProviderCredentialDialog } from '@/app/components/base/modal/create-provider-credential-modal';
import { COMPLETE_PROVIDER } from '@/app/components/providers';
import { useAllProviderCredentials } from '@/hooks/use-model';
import { useProviderContext } from '@/context/provider-context';

interface CredentialDropdownProps {
  className?: string;
  providerId?: string;
  currentCredential?: string;
  onChangeCredential: (credential: VaultCredential) => void;
}

export const CredentialDropdown: FC<CredentialDropdownProps> = props => {
  const [loading] = useState(false);
  const { providerCredentials } = useAllProviderCredentials();
  const ctx = useProviderContext();
  const [createProviderModalOpen, setCreateProviderModalOpen] = useState(false);
  const [currentProviderCredentials, setCurrentProviderCredentials] = useState<
    Array<VaultCredential>
  >([]);

  useEffect(() => {
    setCurrentProviderCredentials(
      providerCredentials.filter(y => y.getVaulttypeid() === props.providerId),
    );
  }, [providerCredentials, props.providerId]);
  const handleSearch = useCallback(
    (q: React.ChangeEvent<HTMLInputElement>) => {
      if (q.target.value && q.target.value.trim() !== '') {
        setCurrentProviderCredentials(
          providerCredentials.filter(
            y =>
              y.getVaulttypeid() === props.providerId &&
              y.getName().includes(q.target.value.trim()),
          ),
        );
      } else {
        setCurrentProviderCredentials(
          providerCredentials.filter(
            y => y.getVaulttypeid() === props.providerId,
          ),
        );
      }
    },
    [providerCredentials, props.providerId],
  );

  return (
    <>
      <CreateProviderCredentialDialog
        modalOpen={createProviderModalOpen}
        setModalOpen={setCreateProviderModalOpen}
        currentProviderId={props.providerId}
      />
      <FieldSet>
        <FormLabel>Credential</FormLabel>
        <div
          className={cn(
            'outline-solid outline-transparent',
            'focus-within:outline-blue-600 focus:outline-blue-600',
            'border-b border-gray-400 dark:border-gray-600',
            'focus-within:border-transparent!',
            'transition-all duration-200 ease-in-out',
            'flex relative',
            'bg-light-background dark:bg-gray-950',
            'pt-px pl-px',
            props.className,
          )}
        >
          <div className="w-full relative">
            <Dropdown
              disable={loading}
              searchable
              className="max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none! outline-hidden"
              currentValue={currentProviderCredentials.find(
                x => x.getId() === props.currentCredential,
              )}
              setValue={(c: VaultCredential) => {
                props.onChangeCredential(c);
              }}
              onSearching={handleSearch}
              allValue={currentProviderCredentials}
              placeholder="Select credential"
              option={(c: VaultCredential) => {
                return (
                  <div className="relative overflow-hidden flex-1 flex flex-row space-x-3">
                    <div className="flex">
                      <span className="inline-flex items-center gap-1.5 sm:gap-2 max-w-full text-sm font-medium">
                        <img
                          alt={
                            COMPLETE_PROVIDER.find(
                              x => x.id === c.getVaulttypeid(),
                            )?.name
                          }
                          loading="lazy"
                          className="w-5 h-5 align-middle block shrink-0"
                          src={
                            COMPLETE_PROVIDER.find(
                              x => x.id === c.getVaulttypeid(),
                            )?.image
                          }
                        />
                        <span className="truncate capitalize">
                          {
                            COMPLETE_PROVIDER.find(
                              x => x.id === c.getVaulttypeid(),
                            )?.name
                          }
                        </span>
                        <span>/</span>
                        <span className="font-medium">{c.getName()}</span>
                      </span>
                      <span className="font-medium ml-4">[{c.getId()}]</span>
                    </div>
                  </div>
                );
              }}
              label={(c: VaultCredential) => {
                return (
                  <div className="relative overflow-hidden flex-1 flex flex-row space-x-3">
                    <div className="flex">
                      <span className="opacity-70">Vault</span>
                      <span className="opacity-70 px-1">/</span>
                      <span className="font-medium">{c.getName()}</span>
                    </div>
                  </div>
                );
              }}
            />
          </div>
          <IButton
            onClick={() => {
              ctx.reloadProviderCredentials();
            }}
          >
            <RotateCcw className={cn('w-4 h-4')} strokeWidth={1.5} />
          </IButton>
          <IButton
            onClick={() => {
              setCreateProviderModalOpen(true);
            }}
          >
            <Plus className={cn('w-4 h-4')} strokeWidth={1.5} />
          </IButton>
        </div>
      </FieldSet>
    </>
  );
};
