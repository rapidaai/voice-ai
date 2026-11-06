import type { ComponentType } from 'react';
import type { Params } from 'react-router-dom';

export interface BreadcrumbLabelProps {
  params: Readonly<Params<string>>;
}

export type BreadcrumbDefinition =
  | { label: string }
  | { render: ComponentType<BreadcrumbLabelProps> };

export interface AppRouteHandle {
  breadcrumb?: BreadcrumbDefinition;
}

export function breadcrumb(label: string): AppRouteHandle {
  return { breadcrumb: { label } };
}
