import { CreateFullPageHeader } from '@/app/components/ui/composites';

interface DeploymentEditHeaderProps {
  eyebrow: string;
  title: string;
  description: string;
}

export function DeploymentEditHeader({
  eyebrow,
  title,
  description,
}: DeploymentEditHeaderProps) {
  return (
    <CreateFullPageHeader
      description={description}
      label={eyebrow}
      title={title}
    />
  );
}
