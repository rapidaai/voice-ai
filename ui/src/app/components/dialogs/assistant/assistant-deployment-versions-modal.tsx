import { useEffect, useState } from 'react';
import {
  Button,
  Loading,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@carbon/react';
import { Copy, Checkmark } from '@carbon/icons-react';
import {
  type AssistantApiDeployment,
  type AssistantDebuggerDeployment,
  type AssistantPhoneDeployment,
  type AssistantWebpluginDeployment,
  type GetAllAssistantApiDeploymentResponse,
  type GetAllAssistantDebuggerDeploymentResponse,
  type GetAllAssistantPhoneDeploymentResponse,
  type GetAllAssistantWebpluginDeploymentResponse,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { ModalProps } from '@/app/components/ui/primitives';
import { RightSideModal } from '@/app/components/dialogs/shared';
import { toHumanReadableDateTime } from '@/utils/date';
import { listAssistantDeploymentVersions } from '@/clients/assistant.client';

export type AssistantDeploymentType = 'debugger' | 'api' | 'web' | 'phone';

interface AssistantDeploymentVersionsModalProps extends ModalProps {
  assistantId: string;
  deploymentType: AssistantDeploymentType | null;
  authId: string;
  token: string;
  projectId: string;
}

type DeploymentVersionRow = {
  id: string;
  createdDate: ReturnType<AssistantApiDeployment['getCreateddate']>;
};

type DeploymentRecord =
  | AssistantApiDeployment
  | AssistantDebuggerDeployment
  | AssistantPhoneDeployment
  | AssistantWebpluginDeployment;

type DeploymentVersionsResponse =
  | GetAllAssistantApiDeploymentResponse
  | GetAllAssistantDebuggerDeploymentResponse
  | GetAllAssistantPhoneDeploymentResponse
  | GetAllAssistantWebpluginDeploymentResponse;

const labelByType: Record<AssistantDeploymentType, string> = {
  api: 'SDK / API',
  debugger: 'Debugger',
  phone: 'Phone Call',
  web: 'Web Widget',
};

export function AssistantDeploymentVersionsModal({
  modalOpen,
  setModalOpen,
  assistantId,
  deploymentType,
  authId,
  token,
  projectId,
}: AssistantDeploymentVersionsModalProps) {
  const [loading, setLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState('');
  const [rows, setRows] = useState<DeploymentVersionRow[]>([]);
  const [copiedVersion, setCopiedVersion] = useState<string | null>(null);

  const title = deploymentType
    ? `${labelByType[deploymentType]} versions`
    : 'Deployment versions';

  useEffect(() => {
    if (!modalOpen || !deploymentType) return;

    let isActive = true;
    setLoading(true);
    setErrorMessage('');
    setRows([]);

    listAssistantDeploymentVersions({
      assistantId,
      deploymentType,
      page: 1,
      pageSize: 100,
      auth: { projectId, token, userId: authId },
    })
      .then(response => {
        if (!isActive) return;
        if (!response?.getSuccess()) {
          setErrorMessage(
            response?.getError?.()?.getHumanmessage?.() ||
              'Unable to load deployment versions.',
          );
          return;
        }

        const mapped: DeploymentVersionRow[] = (
          response as DeploymentVersionsResponse
        )
          .getDataList()
          .map((deployment: DeploymentRecord) => ({
            id: deployment.getId(),
            createdDate: deployment.getCreateddate(),
          }));
        setRows(mapped);
      })
      .catch(() => {
        if (!isActive) return;
        setErrorMessage('Unable to load deployment versions.');
      })
      .finally(() => {
        if (!isActive) return;
        setLoading(false);
      });

    return () => {
      isActive = false;
    };
  }, [modalOpen, deploymentType, assistantId, authId, projectId, token]);

  const copyVersion = (id: string) => {
    const version = `vrsn_${id}`;
    navigator.clipboard.writeText(version);
    setCopiedVersion(version);
    toast.success('Version copied.');
    setTimeout(() => setCopiedVersion(null), 1500);
  };

  return (
    <RightSideModal
      modalOpen={modalOpen}
      setModalOpen={setModalOpen}
      className="w-[580px]"
      label="Deployment"
      title={title}
    >
      <div className="relative flex flex-col flex-1 min-h-0">
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <Loading withOverlay={false} />
          </div>
        ) : errorMessage ? (
          <p className="text-sm text-support-error ">{errorMessage}</p>
        ) : rows.length === 0 ? (
          <p className="text-sm text-muted ">No versions found.</p>
        ) : (
          <div className="overflow-auto flex-1">
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader>Version</TableHeader>
                  <TableHeader>Date</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {rows.map(row => (
                  <TableRow key={row.id}>
                    <TableCell className="!font-mono !text-xs">
                      <span className="inline-flex items-center gap-1">
                        {`vrsn_${row.id}`}
                        <Button
                          hasIconOnly
                          renderIcon={
                            copiedVersion === `vrsn_${row.id}`
                              ? Checkmark
                              : Copy
                          }
                          iconDescription="Copy version id"
                          kind="ghost"
                          size="sm"
                          onClick={() => copyVersion(row.id)}
                          className="!min-h-0 !p-1"
                        />
                      </span>
                    </TableCell>
                    <TableCell className="!text-xs whitespace-nowrap">
                      {row.createdDate
                        ? toHumanReadableDateTime(row.createdDate)
                        : '—'}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </RightSideModal>
  );
}
