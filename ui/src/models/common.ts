export enum ResourceRole {
  // if you are the one who is created this than
  owner = 'owner',

  // if project is same than your are in project
  projectMember = 'project-member',

  // if organization is same than you are memeber
  organizationMember = 'organization-member',

  // if organization is same than you are memeber
  anyone = 'anyone',
}

export enum InputVarType {
  stringInput = 'string',
  textInput = 'text',
  paragraph = 'paragraph',
  select = 'select',
  number = 'number',
  url = 'url',
  files = 'files',
  json = 'json', // obj, array
  contexts = 'contexts', // knowledge retrieval
}

export type InputVar = {
  type: InputVarType;
  label:
    | string
    | {
        nodeName: string;
        variable: string;
      };
  variable: string;
  max_length?: number;
  default?: string;
  required: boolean;
  hint?: string;
  options?: string[];
};
