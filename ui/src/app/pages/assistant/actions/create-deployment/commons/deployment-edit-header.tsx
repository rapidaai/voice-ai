import { Text } from '@/app/components/ui/primitives/text';

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
    <header className="px-6 pb-6 pt-8 md:px-8">
      <Text
        as="p"
        variant="label-01"
        className="uppercase text-[var(--cds-text-secondary)]"
      >
        {eyebrow}
      </Text>
      <Text as="h1" variant="heading-03" className="mt-2">
        {title}
      </Text>
      <Text
        as="p"
        variant="body-01"
        className="mt-2 text-[var(--cds-text-secondary)]"
      >
        {description}
      </Text>
    </header>
  );
}
