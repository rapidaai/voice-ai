import { useTheme } from './theme-provider';

const hasProtocol = (value: string) => /^[a-z][a-z0-9+.-]*:/i.test(value);

const trimEndSlash = (value: string) => value.replace(/\/+$/, '');

const trimStartSlash = (value: string) => value.replace(/^\/+/, '');

export const buildDocumentationUrl = (documentationRoot: string, path = '') => {
  if (!path) return documentationRoot;
  if (hasProtocol(path)) return path;
  if (documentationRoot === '#') return '#';

  return `${trimEndSlash(documentationRoot)}/${trimStartSlash(path)}`;
};

export const useDocumentationUrl = (path = '') => {
  const { theme } = useTheme();
  return buildDocumentationUrl(theme.links.documentation, path);
};
