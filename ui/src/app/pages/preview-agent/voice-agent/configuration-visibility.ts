export function isVisibleConfigurationKey(key: string) {
  const configurationKey = key.toLowerCase();
  return (
    configurationKey !== 'credential' &&
    !configurationKey.endsWith('credential_id')
  );
}
