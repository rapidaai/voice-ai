import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import ReactFlow, {
  addEdge,
  applyEdgeChanges,
  applyNodeChanges,
  Background,
  Handle,
  NodeToolbar,
  Panel,
  Position,
  updateEdge,
  useUpdateNodeInternals,
} from 'reactflow';
import type {
  Connection,
  ConnectionLineComponentProps,
  DefaultEdgeOptions,
  Edge,
  EdgeChange,
  Node,
  NodeChange,
  NodeProps,
  NodeDragHandler,
  ReactFlowInstance,
  SelectionDragHandler,
} from 'reactflow';
import { Helmet } from '@/app/components/app-shell/helmet';
import { ConfigPrompt } from '@/app/components/domain/configuration/config-prompt';
import { FormLabel } from '@/app/components/ui/primitives';
import { CornerBorderOverlay } from '@/app/components/ui/primitives';
import { PrimaryButton, SecondaryButton } from '@/app/components/ui/primitives';
import {
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
} from '@/app/components/ui/primitives';
import { ReactSortable } from 'react-sortablejs';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import toast from 'react-hot-toast/headless';
import {
  GetDefaultTextProviderConfigIfInvalid,
  GetDefaultTextProviderConfigOnProviderSwitch,
  TextProvider,
} from '@/app/components/domain/providers/text';
import {
  Button,
  Checkbox,
  ContentSwitcher,
  OverflowMenu,
  OverflowMenuItem,
  StructuredListBody,
  StructuredListCell,
  StructuredListRow,
  StructuredListWrapper,
  Tag,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
  Select,
  SelectItem,
  Slider,
  Switch,
  TextArea,
  TextInput,
  Tooltip,
  Toggle,
} from '@carbon/react';
import {
  Add,
  Api,
  Bot,
  CaretDown,
  Chat as ChatIcon,
  Checkmark,
  Close,
  Code,
  Document,
  Draggable,
  Edit,
  Flow,
  Folder,
  Information,
  Link,
  Locked,
  MachineLearning,
  Notebook,
  PhoneOff,
  PhoneOutgoing,
  PromptTemplate,
  Save,
  SettingsAdjust,
  SidePanelClose,
  SidePanelOpen,
  TextFont,
  ToolKit,
  TrashCan,
  Unlocked,
  Webhook as WebhookIcon,
} from '@carbon/icons-react';
import { Metadata } from '@rapidaai/react';
import {
  AgentNodeCard,
  ChatInputNodeCard,
  ConditionNodeCard,
  GenericNodeCard,
  StaticMessageNodeCard,
  StickyNoteNodeCard,
} from '@/app/pages/assistant/actions/agentflow/nodes';
import agentflowExamples from '@/prompts/agentflow/index.json';
import { cn } from '@/utils';

export type AgentPromptTemplate = {
  prompt: { role: string; content: string }[];
  variables: { name: string; type: string; defaultvalue: string }[];
};

export type AgentTransitionParameter = {
  id: string;
  name: string;
  type: string;
  description: string;
  required: boolean;
  schema?: Record<string, unknown>;
};

export type AgentTransition = {
  id: string;
  name: string;
  description: string;
  parameters?: AgentTransitionParameter[];
  properties?: Record<string, unknown>;
  required?: string[];
};

export type ChatInputArgument = {
  id: string;
  name: string;
  type: string;
  defaultvalue: string;
};

export type ConditionRule = {
  id: string;
  sourceNodeId: string;
  sourceHandle?: string;
  field: string;
  operator: string;
  value: string;
  left?: string;
};

export type AgentflowNodeType =
  | 'chat-input'
  | 'chat-output'
  | 'message'
  | 'sticky-note'
  | 'text-input'
  | 'text-output'
  | 'api-request'
  | 'url'
  | 'file'
  | 'directory'
  | 'start'
  | 'prompt'
  | 'condition'
  | 'loop'
  | 'router'
  | 'tool'
  | 'mcp'
  | 'knowledge'
  | 'variable'
  | 'webhook'
  | 'action'
  | 'transfer'
  | 'user-input'
  | 'wait'
  | 'end'
  | 'fallback'
  | 'error';

export type AgentflowNode = {
  id: string;
  type: AgentflowNodeType;
  label: string;
  x: number;
  y: number;
  config?: AgentflowNodeConfig;
};

export type AgentflowEdge = Edge;

export type AgentflowNodeConfigValue = string | number | boolean;

export type AgentflowNodeConfig = Record<string, AgentflowNodeConfigValue>;

export type AgentflowDefinitionConfigValue =
  | AgentflowNodeConfigValue
  | null
  | AgentflowDefinitionConfigValue[]
  | { [key: string]: AgentflowDefinitionConfigValue };

export type AgentflowDefinitionNodeConfig = Record<
  string,
  AgentflowDefinitionConfigValue
>;

export type AgentflowFieldType =
  | 'text'
  | 'textarea'
  | 'select'
  | 'number'
  | 'toggle';

export type AgentflowField = {
  name: string;
  label: string;
  description: string;
  type: AgentflowFieldType;
  section: 'basic' | 'advanced';
  defaultValue: AgentflowNodeConfigValue;
  options?: string[];
  optional?: boolean;
  min?: number;
  max?: number;
  step?: number;
  control?: 'input' | 'slider-input';
};

export type AgentflowNodeTemplate = {
  type: AgentflowNodeType;
  group?: AgentflowGroupKey;
  label: string;
  eyebrow: string;
  description: string;
  inputs: string[];
  outputs: string[];
  options: string[];
  icon: React.ComponentType<{ size?: number; className?: string }>;
  disabled?: boolean;
  hiddenInSidebar?: boolean;
};

type AgentflowGroupKey =
  | 'input-output'
  | 'data-sources'
  | 'models-agents'
  | 'llm-operations'
  | 'files-knowledge'
  | 'processing'
  | 'flow-control'
  | 'utilities';

type AgentflowPaletteGroup = {
  key: string;
  title: string;
  icon: React.ComponentType<{ size?: number; className?: string }>;
  nodeTypes: AgentflowNodeType[];
};

type AgentflowReactNodeData = {
  node: AgentflowNode;
  onOpenDetails: (node: AgentflowNode) => void;
  onSave: () => void;
  onDuplicate: (node: AgentflowNode) => void;
  onDelete: (node: AgentflowNode) => void;
  onUpdateConfig: (
    nodeId: string,
    name: string,
    value: AgentflowNodeConfigValue,
  ) => void;
  onUpdateConfigValues: (nodeId: string, values: AgentflowNodeConfig) => void;
  onAddTransition: (
    event: React.MouseEvent<HTMLButtonElement>,
    node: AgentflowNode,
  ) => void;
};

type ConditionFieldOption = {
  value: string;
  label: string;
};

type ConditionSourceOption = {
  value: string;
  label: string;
  node: AgentflowNode;
  sourceHandle?: string;
};

type AgentflowStoredNodeData = {
  node: AgentflowNode;
};

type AgentflowStoredNode = Node<AgentflowStoredNodeData>;

export type AgentflowDefinitionNode = {
  id: string;
  type: AgentflowNodeType;
  label: string;
  position: {
    x: number;
    y: number;
  };
  config?: AgentflowDefinitionNodeConfig;
};

export type AgentflowDefinitionEdge = {
  id: string;
  source: string;
  sourceHandle?: string;
  target: string;
  targetHandle?: string;
};

export type AgentflowDefinition = {
  schemaVersion: '2026-07-06';
  entryNodeId: string;
  name?: string;
  description?: string;
  tags?: string[];
  nodes: AgentflowDefinitionNode[];
  edges: AgentflowDefinitionEdge[];
  viewport?: {
    x: number;
    y: number;
    zoom: number;
  };
};

const exampleAgentflowDefinitions =
  agentflowExamples as unknown as AgentflowDefinition[];

type AgentflowValidationIssue = {
  id: string;
  message: string;
  nodeId?: string;
  edgeId?: string;
};

const NODE_WIDTH = 320;
const STICKY_NOTE_WIDTH = 560;
const FLOW_START_X = 88;
const FLOW_NODE_GAP = 120;
const FLOW_NODE_STEP = NODE_WIDTH + FLOW_NODE_GAP;
const FLOW_CONNECTOR_BASELINE_Y = 176;
const COMPACT_NODE_HEADER_HEIGHT = 48;
const CHAT_INPUT_NODE_HEADER_HEIGHT = COMPACT_NODE_HEADER_HEIGHT;
const CHAT_INPUT_ARGUMENT_HEADER_HEIGHT = 48;
const CHAT_INPUT_ARGUMENT_ROW_HEIGHT = 36;
const CHAT_INPUT_NODE_BOTTOM_PADDING = 16;
const AGENT_NODE_INPUT_PORT_TOP = 28;
const AGENT_NODE_PROMPT_TOP = 64;
const AGENT_NODE_PROMPT_HEIGHT = 132;
const AGENT_NODE_TRANSITION_HEADER_HEIGHT = 48;
const AGENT_NODE_TRANSITION_ROW_HEIGHT = 40;
const AGENT_NODE_BOTTOM_PADDING = 0;
const AGENT_NODE_TRANSITION_LIST_TOP =
  AGENT_NODE_PROMPT_TOP +
  AGENT_NODE_PROMPT_HEIGHT +
  AGENT_NODE_TRANSITION_HEADER_HEIGHT;
const GENERIC_NODE_HEADER_HEIGHT = 48;
const GENERIC_NODE_SECTION_TOP = 16;
const GENERIC_NODE_SECTION_LABEL_HEIGHT = 18;
const GENERIC_NODE_SECTION_LABEL_BOTTOM_MARGIN = 8;
const GENERIC_NODE_SETTING_ROW_HEIGHT = 40;
const GENERIC_NODE_CONNECTION_SECTION_TOP = 16;
const GENERIC_NODE_CONNECTION_HEADER_HEIGHT = 48;
const GENERIC_NODE_CONNECTION_ROW_HEIGHT = 36;
const GENERIC_NODE_BOTTOM_PADDING = 16;
const CONDITION_NODE_INPUT_PORT_TOP = 28;
const CONDITION_NODE_CONDITION_HEADER_HEIGHT = 48;
const CONDITION_NODE_ROW_HEIGHT = 40;
const CONDITION_NODE_BOTTOM_PADDING = 16;
const DEFAULT_AGENT_PROMPT =
  'Hello this is customer support department,\nhow can I help you today?';
const DEFAULT_AGENT_TRANSITIONS: AgentTransition[] = [
  {
    id: 'transition-return-package',
    name: 'return_package',
    description:
      'Use this when the caller wants to return an item or asks about return eligibility.',
    parameters: [],
  },
  {
    id: 'transition-order-status',
    name: 'check_order_status',
    description:
      'Use this when the caller wants to check the status of an existing order.',
    parameters: [
      {
        id: 'parameter-order-id',
        name: 'order_id',
        type: 'string',
        description: 'The caller order id.',
        required: true,
      },
    ],
  },
  {
    id: 'transition-fallback',
    name: 'fallback',
    description:
      'Use this when the caller request does not match another transition.',
    parameters: [],
  },
];
const DEFAULT_CHAT_INPUT_ARGUMENTS: ChatInputArgument[] = [];
const CHAT_INPUT_ARGUMENT_TYPES = ['string', 'number', 'boolean', 'object'];
const CONDITION_OPERATOR_OPTIONS = [
  'equals',
  'not equals',
  'contains',
  'greater than',
  'less than',
  'exists',
  'is true',
  'is false',
];
const DEFAULT_CONDITION_RULES: ConditionRule[] = [];
const FUNCTION_PARAMETER_TYPE_OPTIONS = [
  { value: 'string', label: 'String' },
  { value: 'integer', label: 'Integer' },
  { value: 'number', label: 'Number' },
  { value: 'boolean', label: 'Boolean' },
  { value: 'object', label: 'Object' },
  { value: 'array', label: 'Array' },
];
const FUNCTION_PARAMETER_TYPE_VALUES = new Set(
  FUNCTION_PARAMETER_TYPE_OPTIONS.map(option => option.value),
);
const DEFAULT_AGENT_PROVIDER = 'azure-foundry';
const NODE_TOOLBAR_CLASS =
  'absolute z-20 flex h-8 items-center gap-0 rounded-none border border-gray-200 bg-white p-0 shadow-sm dark:border-gray-800 dark:bg-gray-900';
const NODE_TOOLBAR_BUTTON_CLASS =
  '!inline-flex !h-7 !min-h-7 !items-center !gap-1.5 !px-2 !text-xs !font-normal !text-gray-800 hover:!bg-gray-100 hover:!text-gray-900 dark:!text-gray-100 dark:hover:!bg-gray-800 [&_svg]:!fill-current';
const NODE_TOOLBAR_MENU_CLASS =
  '!h-7 !w-7 !text-gray-800 hover:!bg-gray-100 dark:!text-gray-100 dark:hover:!bg-gray-800 [&_svg]:!fill-current';
const NODE_TOOLBAR_MENU_ITEM_CLASS =
  '!text-xs !text-gray-900 dark:!text-gray-100';
const INPUT_CONNECTOR_CLASS = 'z-20 h-2.5 w-2.5 rounded-full border-none';
const OUTPUT_CONNECTOR_CLASS = 'z-20 h-2.5 w-2.5 rounded-full border-none';
const INPUT_CONNECTOR_COLORS = ['#6b7280'];
const OUTPUT_CONNECTOR_COLORS = ['#0f62fe'];
const EDGE_STYLE = {
  stroke: '#6b7280',
  strokeWidth: 1.5,
};
const EDGE_INTERACTION_WIDTH = 24;

const paletteGroups: AgentflowPaletteGroup[] = [
  {
    key: 'input-output',
    title: 'Input and Output',
    icon: ChatIcon,
    nodeTypes: ['chat-input'],
  },
  {
    key: 'agent',
    title: 'Agent',
    icon: Bot,
    nodeTypes: ['prompt'],
  },
  {
    key: 'flow-control',
    title: 'Flow Control',
    icon: Flow,
    nodeTypes: ['condition'],
  },
  {
    key: 'utilities',
    title: 'Utility',
    icon: ToolKit,
    nodeTypes: ['message', 'sticky-note'],
  },
  {
    key: 'action',
    title: 'Action',
    icon: PhoneOutgoing,
    nodeTypes: ['end', 'transfer'],
  },
];

const NODE_DRAG_TYPE = 'application/rapida-agentflow-node';
const START_NODE_ID = 'chat-input-1';
const MIN_CANVAS_ZOOM = 0.25;
const DEFAULT_CANVAS_ZOOM = 0.75;
const MAX_CANVAS_ZOOM = 2;
const CANVAS_VIEWPORT_DURATION = 180;

const nodeTemplates: AgentflowNodeTemplate[] = [
  {
    type: 'chat-input',
    group: 'input-output',
    label: 'Chat Input',
    eyebrow: 'Start',
    description: 'Begin the workflow from the inbound caller message.',
    inputs: [],
    outputs: ['next'],
    options: ['Arguments'],
    icon: ChatIcon,
  },
  {
    type: 'chat-output',
    group: 'input-output',
    label: 'Chat Output',
    eyebrow: 'Output',
    description: 'Speak or emit the current assistant response to the caller.',
    inputs: ['message'],
    outputs: ['next'],
    options: ['Message'],
    icon: ChatIcon,
  },
  {
    type: 'message',
    group: 'utilities',
    label: 'Static Message',
    eyebrow: 'Utility',
    description: 'Emit a fixed response without calling an agent model.',
    inputs: ['incoming'],
    outputs: ['response'],
    options: ['Message'],
    icon: ChatIcon,
  },
  {
    type: 'sticky-note',
    group: 'utilities',
    label: 'Sticky Note',
    eyebrow: 'Note',
    description: 'Add a non-runtime annotation to document the flow.',
    inputs: [],
    outputs: [],
    options: ['Note'],
    icon: Notebook,
  },
  {
    type: 'text-input',
    group: 'input-output',
    label: 'Text Input',
    eyebrow: 'Input',
    description: 'Accept text as a workflow input variable.',
    inputs: [],
    outputs: ['text'],
    options: ['Text'],
    icon: TextFont,
  },
  {
    type: 'text-output',
    group: 'input-output',
    label: 'Text Output',
    eyebrow: 'Output',
    description: 'Return text from the workflow.',
    inputs: ['text'],
    outputs: [],
    options: ['Text'],
    icon: TextFont,
  },
  {
    type: 'start',
    group: 'input-output',
    label: 'Start',
    eyebrow: 'Entry',
    description: 'Entry point for a published assistant workflow.',
    inputs: [],
    outputs: ['next'],
    options: ['Entry node', 'Published flow'],
    icon: Bot,
    hiddenInSidebar: true,
  },
  {
    type: 'api-request',
    group: 'data-sources',
    label: 'API Request',
    eyebrow: 'HTTP',
    description: 'Make HTTP requests to one or more URLs and return data.',
    inputs: ['query params', 'body'],
    outputs: ['data', 'error'],
    options: ['URLs', 'Method', 'Headers'],
    icon: Api,
  },
  {
    type: 'url',
    group: 'data-sources',
    label: 'URL',
    eyebrow: 'Loader',
    description: 'Fetch page content from one or more URLs.',
    inputs: ['urls'],
    outputs: ['data'],
    options: ['URLs'],
    icon: Link,
  },
  {
    type: 'file',
    group: 'data-sources',
    label: 'File',
    eyebrow: 'Loader',
    description: 'Load text data from a file path or uploaded file.',
    inputs: ['path'],
    outputs: ['data'],
    options: ['Path', 'Silent errors'],
    icon: Document,
  },
  {
    type: 'directory',
    group: 'data-sources',
    label: 'Directory',
    eyebrow: 'Loader',
    description: 'Recursively load files from a directory.',
    inputs: ['path'],
    outputs: ['data'],
    options: ['Path', 'Types', 'Depth'],
    icon: Folder,
  },
  {
    type: 'prompt',
    group: 'llm-operations',
    label: 'Agent Node',
    eyebrow: 'Agent',
    description:
      'Define an agent turn with prompt text, tools, and transition conditions.',
    inputs: ['incoming'],
    outputs: ['transition'],
    options: ['Prompt', 'Transitions', 'Tools'],
    icon: Bot,
  },
  {
    type: 'condition',
    group: 'flow-control',
    label: 'If / Else',
    eyebrow: 'Branch',
    description:
      'Deterministically branch using an argument, transition parameter, variable, or tool result.',
    inputs: ['value'],
    outputs: ['true', 'false'],
    options: ['Source', 'Operator', 'Compare'],
    icon: PromptTemplate,
  },
  {
    type: 'loop',
    group: 'flow-control',
    label: 'Loop',
    eyebrow: 'Repeat',
    description:
      'Repeat a branch over a list, retry condition, or bounded conversation task.',
    inputs: ['items'],
    outputs: ['next item', 'done'],
    options: ['Items', 'Limit', 'Stop condition'],
    icon: PromptTemplate,
  },
  {
    type: 'router',
    group: 'flow-control',
    label: 'Router',
    eyebrow: 'Route',
    description: 'Route by intent, variable, tool result, or fallback.',
    inputs: ['input'],
    outputs: ['route', 'fallback'],
    options: ['Routes', 'Priority'],
    icon: PromptTemplate,
  },
  {
    type: 'tool',
    group: 'utilities',
    label: 'Tool',
    eyebrow: 'Action',
    description: 'Call a configured assistant tool.',
    inputs: ['arguments'],
    outputs: ['result', 'error'],
    options: ['Tool', 'Mapping'],
    icon: ToolKit,
  },
  {
    type: 'mcp',
    group: 'models-agents',
    label: 'MCP',
    eyebrow: 'Tool',
    description: 'Invoke an MCP-backed tool or server capability.',
    inputs: ['request'],
    outputs: ['result', 'error'],
    options: ['Server', 'Tool'],
    icon: MachineLearning,
  },
  {
    type: 'knowledge',
    group: 'files-knowledge',
    label: 'Knowledge',
    eyebrow: 'Retrieval',
    description: 'Retrieve context from configured knowledge sources.',
    inputs: ['query'],
    outputs: ['documents', 'empty'],
    options: ['Source', 'Top K'],
    icon: PromptTemplate,
  },
  {
    type: 'variable',
    group: 'processing',
    label: 'Variable',
    eyebrow: 'State',
    description: 'Set, merge, or clear workflow runtime variables.',
    inputs: ['value'],
    outputs: ['updated'],
    options: ['Variable', 'Operation'],
    icon: PromptTemplate,
  },
  {
    type: 'webhook',
    group: 'input-output',
    label: 'Webhook',
    eyebrow: 'HTTP',
    description: 'Call an external webhook from the workflow.',
    inputs: ['payload'],
    outputs: ['response', 'error'],
    options: ['URL', 'Headers'],
    icon: WebhookIcon,
    disabled: true,
  },
  {
    type: 'action',
    group: 'utilities',
    label: 'Run Action',
    eyebrow: 'Function',
    description: 'Run a configured tool, function, or platform action.',
    inputs: ['payload'],
    outputs: ['done', 'error'],
    options: ['Function', 'Payload'],
    icon: ToolKit,
  },
  {
    type: 'transfer',
    group: 'utilities',
    label: 'Transfer Call',
    eyebrow: 'Action',
    description: 'Transfer the active call to phone numbers or SIP URIs.',
    inputs: ['call'],
    outputs: [],
    options: ['Transfer destinations'],
    icon: PhoneOutgoing,
  },
  {
    type: 'user-input',
    group: 'input-output',
    label: 'User Input',
    eyebrow: 'Wait',
    description: 'Pause progression until the next user turn arrives.',
    inputs: ['prompt'],
    outputs: ['message', 'timeout'],
    options: ['Prompt', 'Timeout'],
    icon: Bot,
    hiddenInSidebar: true,
  },
  {
    type: 'wait',
    group: 'processing',
    label: 'Wait',
    eyebrow: 'Delay',
    description: 'Hold the workflow for a configured duration or signal.',
    inputs: ['start'],
    outputs: ['elapsed'],
    options: ['Duration', 'Signal'],
    icon: PromptTemplate,
  },
  {
    type: 'end',
    group: 'utilities',
    label: 'End Conversation',
    eyebrow: 'Action',
    description: 'End the active conversation.',
    inputs: ['call'],
    outputs: [],
    options: [],
    icon: PhoneOff,
  },
  {
    type: 'fallback',
    group: 'flow-control',
    label: 'Fallback',
    eyebrow: 'Recovery',
    description: 'Handle unmatched conditions or failed transitions.',
    inputs: ['event'],
    outputs: ['recover', 'end'],
    options: ['Policy', 'Message'],
    icon: PromptTemplate,
  },
  {
    type: 'error',
    group: 'flow-control',
    label: 'Error',
    eyebrow: 'Failure',
    description: 'Handle tool, model, or workflow execution failures.',
    inputs: ['error'],
    outputs: ['retry', 'end'],
    options: ['Retry', 'Escalate'],
    icon: Code,
  },
];

const nodeEditFields: Record<AgentflowNodeType, AgentflowField[]> = {
  'chat-input': [
    {
      name: 'arguments',
      label: 'Arguments',
      description:
        'Input arguments exposed by the start node when the workflow begins.',
      type: 'textarea',
      section: 'basic',
      defaultValue: JSON.stringify(DEFAULT_CHAT_INPUT_ARGUMENTS),
    },
  ],
  'chat-output': [
    {
      name: 'response_key',
      label: 'Response key',
      description: 'Runtime key that contains the response to speak or emit.',
      type: 'text',
      section: 'basic',
      defaultValue: 'response',
    },
    {
      name: 'format',
      label: 'Format',
      description: 'How the response is emitted to the channel.',
      type: 'select',
      section: 'basic',
      defaultValue: 'text',
      options: ['text', 'markdown', 'json'],
    },
  ],
  message: [
    {
      name: 'message',
      label: 'Message',
      description: 'Fixed message emitted by this node.',
      type: 'textarea',
      section: 'basic',
      defaultValue: 'Thanks for calling. One moment while I help you.',
    },
    {
      name: 'post_delay_ms',
      label: 'Post delay ms',
      description:
        'Milliseconds to wait after the message is spoken before continuing.',
      type: 'number',
      section: 'basic',
      defaultValue: 0,
      min: 0,
      max: 30000,
      step: 100,
      control: 'slider-input',
    },
  ],
  'sticky-note': [
    {
      name: 'note',
      label: 'Note',
      description: 'Canvas annotation text. It does not affect runtime.',
      type: 'textarea',
      section: 'basic',
      defaultValue: '# Quick start',
      optional: true,
    },
    {
      name: 'color',
      label: 'Color',
      description: 'Visual color used for this note on the canvas.',
      type: 'select',
      section: 'basic',
      defaultValue: 'yellow',
      options: ['yellow', 'blue', 'green', 'purple'],
      optional: true,
    },
  ],
  'text-input': [
    {
      name: 'variable',
      label: 'Variable name',
      description: 'Name of the text input variable exposed by this node.',
      type: 'text',
      section: 'basic',
      defaultValue: 'text',
    },
    {
      name: 'required',
      label: 'Required',
      description: 'Require the value before the flow can continue.',
      type: 'toggle',
      section: 'advanced',
      defaultValue: true,
    },
  ],
  'text-output': [
    {
      name: 'value',
      label: 'Value',
      description: 'Text value or variable reference to return.',
      type: 'textarea',
      section: 'basic',
      defaultValue: '{{ text }}',
    },
    {
      name: 'trim',
      label: 'Trim whitespace',
      description: 'Remove leading and trailing whitespace before output.',
      type: 'toggle',
      section: 'advanced',
      defaultValue: true,
    },
  ],
  start: [
    {
      name: 'entry_mode',
      label: 'Entry mode',
      description: 'How the workflow is triggered.',
      type: 'select',
      section: 'basic',
      defaultValue: 'published assistant',
      options: ['published assistant', 'test run', 'webhook'],
    },
    {
      name: 'trace',
      label: 'Trace run',
      description: 'Capture runtime trace data from the start node.',
      type: 'toggle',
      section: 'advanced',
      defaultValue: true,
    },
  ],
  'api-request': [
    {
      name: 'urls',
      label: 'URLs',
      description: 'One or more request URLs, separated by new lines.',
      type: 'textarea',
      section: 'basic',
      defaultValue: 'https://api.example.com',
    },
    {
      name: 'method',
      label: 'Method',
      description: 'HTTP method used for the request.',
      type: 'select',
      section: 'basic',
      defaultValue: 'GET',
      options: ['GET', 'POST', 'PATCH', 'PUT'],
    },
    {
      name: 'headers',
      label: 'Headers',
      description: 'JSON headers sent with the request.',
      type: 'textarea',
      section: 'basic',
      defaultValue: '{\n  "Content-Type": "application/json"\n}',
    },
    {
      name: 'body',
      label: 'Body',
      description: 'JSON request body for POST, PATCH, or PUT.',
      type: 'textarea',
      section: 'advanced',
      defaultValue: '{}',
    },
    {
      name: 'timeout',
      label: 'Timeout seconds',
      description: 'Maximum time to wait for the HTTP response.',
      type: 'number',
      section: 'advanced',
      defaultValue: 5,
    },
  ],
  url: [
    {
      name: 'urls',
      label: 'URLs',
      description: 'One or more URLs to fetch, separated by new lines.',
      type: 'textarea',
      section: 'basic',
      defaultValue: 'https://example.com',
    },
    {
      name: 'encoding',
      label: 'Encoding',
      description: 'Text encoding used while reading fetched pages.',
      type: 'select',
      section: 'advanced',
      defaultValue: 'utf-8',
      options: ['utf-8', 'ascii', 'latin-1'],
    },
  ],
  file: [
    {
      name: 'path',
      label: 'Path',
      description: 'Path to the file to load.',
      type: 'text',
      section: 'basic',
      defaultValue: '/path/to/file.txt',
    },
    {
      name: 'silent_errors',
      label: 'Silent errors',
      description: 'Return empty data instead of raising file parsing errors.',
      type: 'toggle',
      section: 'advanced',
      defaultValue: false,
    },
  ],
  directory: [
    {
      name: 'path',
      label: 'Path',
      description: 'Directory path to scan for files.',
      type: 'text',
      section: 'basic',
      defaultValue: '/path/to/directory',
    },
    {
      name: 'types',
      label: 'Types',
      description: 'Optional file extensions to include.',
      type: 'text',
      section: 'basic',
      defaultValue: 'txt,md,pdf',
    },
    {
      name: 'depth',
      label: 'Depth',
      description: 'Directory depth to search.',
      type: 'number',
      section: 'basic',
      defaultValue: 0,
    },
    {
      name: 'recursive',
      label: 'Recursive',
      description: 'Search nested directories recursively.',
      type: 'toggle',
      section: 'advanced',
      defaultValue: true,
    },
    {
      name: 'load_hidden',
      label: 'Load hidden',
      description: 'Include hidden files in the directory scan.',
      type: 'toggle',
      section: 'advanced',
      defaultValue: false,
    },
  ],
  prompt: [
    {
      name: 'prompt',
      label: 'Agent message',
      description: 'Message or instruction shown on the agent node.',
      type: 'textarea',
      section: 'basic',
      defaultValue: DEFAULT_AGENT_PROMPT,
    },
    {
      name: 'transitions',
      label: 'Transitions',
      description: 'One transition condition per line.',
      type: 'textarea',
      section: 'basic',
      defaultValue: JSON.stringify(DEFAULT_AGENT_TRANSITIONS),
    },
  ],
  condition: [
    {
      name: 'conditions',
      label: 'Conditions',
      description:
        'Ordered If conditions. Use argument.size or node_name.tool_name.parameter.',
      type: 'textarea',
      section: 'basic',
      defaultValue: JSON.stringify(DEFAULT_CONDITION_RULES),
    },
  ],
  loop: [
    {
      name: 'items',
      label: 'Items',
      description:
        'List, variable, or expression that provides the items to iterate over.',
      type: 'textarea',
      section: 'basic',
      defaultValue: '{{ state.items }}',
    },
    {
      name: 'max_iterations',
      label: 'Max iterations',
      description: 'Maximum number of loop iterations before exiting.',
      type: 'number',
      section: 'basic',
      defaultValue: 3,
    },
    {
      name: 'stop_condition',
      label: 'Stop condition',
      description: 'Optional condition that exits the loop early.',
      type: 'textarea',
      section: 'advanced',
      defaultValue: '{{ loop.index }} >= 3',
    },
  ],
  router: [
    {
      name: 'route_by',
      label: 'Route by',
      description: 'Value used to choose the next route.',
      type: 'select',
      section: 'basic',
      defaultValue: 'intent',
      options: ['intent', 'variable', 'tool result'],
    },
    {
      name: 'routes',
      label: 'Routes',
      description: 'Ordered route labels and matching values.',
      type: 'textarea',
      section: 'basic',
      defaultValue: 'sales\nsupport\nfallback',
    },
  ],
  tool: [
    {
      name: 'tool_name',
      label: 'Tool',
      description: 'Configured assistant tool to call.',
      type: 'select',
      section: 'basic',
      defaultValue: 'lookup',
      options: ['lookup', 'crm action', 'notification'],
    },
    {
      name: 'arguments',
      label: 'Arguments',
      description: 'JSON argument mapping passed to the tool.',
      type: 'textarea',
      section: 'basic',
      defaultValue: '{\n  "query": "{{ user.message }}"\n}',
    },
    {
      name: 'timeout',
      label: 'Timeout seconds',
      description: 'Maximum time to wait for the tool response.',
      type: 'number',
      section: 'advanced',
      defaultValue: 30,
    },
  ],
  mcp: [
    {
      name: 'server',
      label: 'Server',
      description: 'MCP server used by this node.',
      type: 'text',
      section: 'basic',
      defaultValue: 'default',
    },
    {
      name: 'capability',
      label: 'Capability',
      description: 'Tool or capability exposed by the MCP server.',
      type: 'text',
      section: 'basic',
      defaultValue: 'search',
    },
  ],
  knowledge: [
    {
      name: 'source',
      label: 'Source',
      description: 'Knowledge source or collection to retrieve from.',
      type: 'text',
      section: 'basic',
      defaultValue: 'assistant knowledge',
    },
    {
      name: 'top_k',
      label: 'Top K',
      description: 'Maximum number of chunks to retrieve.',
      type: 'number',
      section: 'advanced',
      defaultValue: 5,
    },
  ],
  variable: [
    {
      name: 'variable',
      label: 'Variable',
      description: 'Runtime variable name to update.',
      type: 'text',
      section: 'basic',
      defaultValue: 'state.value',
    },
    {
      name: 'operation',
      label: 'Operation',
      description: 'How the variable should be updated.',
      type: 'select',
      section: 'basic',
      defaultValue: 'set',
      options: ['set', 'merge', 'append', 'clear'],
    },
  ],
  webhook: [
    {
      name: 'url',
      label: 'URL',
      description: 'External webhook endpoint.',
      type: 'text',
      section: 'basic',
      defaultValue: 'https://example.com/webhook',
    },
    {
      name: 'method',
      label: 'Method',
      description: 'HTTP method used for the webhook call.',
      type: 'select',
      section: 'basic',
      defaultValue: 'POST',
      options: ['POST', 'PUT', 'PATCH'],
    },
  ],
  action: [
    {
      name: 'action',
      label: 'Function',
      description: 'Tool, function, or platform action to execute.',
      type: 'select',
      section: 'basic',
      defaultValue: 'custom function',
      options: ['custom function', 'emit event', 'add tag', 'notify'],
    },
    {
      name: 'payload',
      label: 'Payload',
      description: 'Payload sent to the selected function or action.',
      type: 'textarea',
      section: 'basic',
      defaultValue: '{\n  "event": "agentflow.action"\n}',
    },
  ],
  transfer: [
    {
      name: 'transfer_to',
      label: 'Transfer destinations',
      description:
        'Phone numbers or SIP URIs to transfer calls to. Add one destination per line.',
      type: 'textarea',
      section: 'basic',
      defaultValue: '',
    },
    {
      name: 'transfer_message',
      label: 'Transfer message',
      description: 'Optional message played when transferring the call.',
      type: 'textarea',
      section: 'basic',
      defaultValue: '',
      optional: true,
    },
    {
      name: 'post_transfer_action',
      label: 'Post transfer action',
      description: 'Behavior after transfer completes or fails.',
      type: 'select',
      section: 'basic',
      defaultValue: 'end_call',
      options: ['end_call', 'resume_ai'],
    },
    {
      name: 'ringtone',
      label: 'Ringtone',
      description: 'Ringtone played during the transfer flow.',
      type: 'select',
      section: 'basic',
      defaultValue: 'default',
      options: ['default', 'dial-and-ring', 'ring-ring', 'transfer-music'],
    },
    {
      name: 'transfer_delay',
      label: 'Transfer delay ms',
      description: 'Wait time before starting the transfer flow.',
      type: 'number',
      section: 'basic',
      defaultValue: 500,
    },
  ],
  'user-input': [
    {
      name: 'prompt',
      label: 'Prompt',
      description: 'Message shown before waiting for user input.',
      type: 'textarea',
      section: 'basic',
      defaultValue: 'Can you share a little more detail?',
    },
    {
      name: 'timeout',
      label: 'Timeout seconds',
      description: 'Maximum time to wait for user input.',
      type: 'number',
      section: 'advanced',
      defaultValue: 300,
    },
  ],
  wait: [
    {
      name: 'duration',
      label: 'Duration seconds',
      description: 'Delay before the workflow continues.',
      type: 'number',
      section: 'basic',
      defaultValue: 10,
    },
    {
      name: 'resume_signal',
      label: 'Resume signal',
      description: 'Optional signal that can resume the workflow early.',
      type: 'text',
      section: 'advanced',
      defaultValue: '',
    },
  ],
  end: [],
  fallback: [
    {
      name: 'message',
      label: 'Message',
      description: 'Fallback message or recovery instruction.',
      type: 'textarea',
      section: 'basic',
      defaultValue: 'I could not route this request. Let me try another path.',
    },
    {
      name: 'policy',
      label: 'Policy',
      description: 'Recovery policy for unmatched routes.',
      type: 'select',
      section: 'advanced',
      defaultValue: 'continue',
      options: ['continue', 'retry', 'end'],
    },
  ],
  error: [
    {
      name: 'retry_count',
      label: 'Retry count',
      description: 'Number of automatic retries before escalation.',
      type: 'number',
      section: 'basic',
      defaultValue: 1,
    },
    {
      name: 'escalate',
      label: 'Escalate',
      description: 'Escalate the failure when retries are exhausted.',
      type: 'toggle',
      section: 'advanced',
      defaultValue: true,
    },
  ],
};

const createStoredNode = (node: AgentflowNode): AgentflowStoredNode => ({
  id: node.id,
  type: 'agentflowNode',
  position: { x: node.x, y: node.y },
  style: getStoredNodeStyle(node),
  data: { node },
  deletable: node.id !== START_NODE_ID,
});

const getAgentflowNodeFromStoredNode = (
  reactFlowNode: AgentflowStoredNode,
): AgentflowNode => ({
  ...reactFlowNode.data.node,
  x: reactFlowNode.position.x,
  y: reactFlowNode.position.y,
});

const syncStoredNodeData = (
  reactFlowNode: AgentflowStoredNode,
): AgentflowStoredNode => ({
  ...reactFlowNode,
  data: {
    node: getAgentflowNodeFromStoredNode(reactFlowNode),
  },
});

const getNodeTemplate = (type: AgentflowNodeType) =>
  nodeTemplates.find(template => template.type === type) ?? nodeTemplates[1];

const getNodeFields = (type: AgentflowNodeType) => nodeEditFields[type] ?? [];

const isStartNode = (node: AgentflowNode) => node.id === START_NODE_ID;

const createDefaultConfig = (type: AgentflowNodeType): AgentflowNodeConfig =>
  getNodeFields(type).reduce<AgentflowNodeConfig>((config, field) => {
    config[field.name] = field.defaultValue;
    return config;
  }, {});

const getNodeConfig = (node: AgentflowNode): AgentflowNodeConfig => ({
  ...createDefaultConfig(node.type),
  ...(node.config ?? {}),
});

const normalizeDefinitionConfigValue = (
  value: AgentflowDefinitionConfigValue,
): AgentflowNodeConfigValue => {
  if (value === null) {
    return '';
  }

  if (typeof value === 'object') {
    return JSON.stringify(value);
  }

  return value;
};

const parseDefinitionMetadataList = (
  value: AgentflowDefinitionConfigValue | AgentflowNodeConfigValue | undefined,
) => {
  const rawValue =
    typeof value === 'string'
      ? value
      : value === undefined
        ? ''
        : JSON.stringify(value);

  if (!rawValue.trim()) {
    return [];
  }

  try {
    const parsed = JSON.parse(rawValue) as { key?: string; value?: string }[];

    if (!Array.isArray(parsed)) {
      return [];
    }

    return parsed
      .filter(item => item.key)
      .map(item => ({
        key: String(item.key),
        value: String(item.value ?? ''),
      }));
  } catch {
    return [];
  }
};

const normalizeDefinitionConfig = (
  type: AgentflowNodeType,
  config: AgentflowDefinitionNodeConfig | undefined,
): AgentflowNodeConfig => {
  const normalized = Object.entries(config ?? {}).reduce<AgentflowNodeConfig>(
    (result, [key, value]) => {
      result[key] = normalizeDefinitionConfigValue(value);
      return result;
    },
    {},
  );

  if (type !== 'prompt') {
    return normalized;
  }

  if ('temperature' in normalized) {
    const parameters = parseDefinitionMetadataList(normalized.model_parameters);
    const temperature = String(normalized.temperature);
    const existingTemperature = parameters.find(
      parameter => parameter.key === 'model.temperature',
    );

    if (existingTemperature) {
      existingTemperature.value = temperature;
    } else {
      parameters.push({ key: 'model.temperature', value: temperature });
    }

    normalized.model_parameters = JSON.stringify(parameters);
    delete normalized.temperature;
  }

  delete normalized.model;

  return normalized;
};

const getSerializableNodeConfig = (node: AgentflowNode): AgentflowNodeConfig =>
  normalizeDefinitionConfig(node.type, getNodeConfig(node));

export const createAgentflowDefinition = (
  nodes: AgentflowNode[],
  edges: AgentflowEdge[],
  viewport?: AgentflowDefinition['viewport'],
  metadata?: Pick<AgentflowDefinition, 'name' | 'description' | 'tags'>,
): AgentflowDefinition => ({
  schemaVersion: '2026-07-06',
  entryNodeId: START_NODE_ID,
  ...(metadata?.name?.trim() ? { name: metadata.name.trim() } : {}),
  ...(metadata?.description?.trim()
    ? { description: metadata.description.trim() }
    : {}),
  ...(metadata?.tags?.length ? { tags: metadata.tags } : {}),
  nodes: nodes.map(node => ({
    id: node.id,
    type: node.type,
    label: node.label,
    position: {
      x: node.x,
      y: node.y,
    },
    config: getSerializableNodeConfig(node),
  })),
  edges: edges.map(edge => ({
    id: edge.id,
    source: edge.source,
    sourceHandle: edge.sourceHandle ?? undefined,
    target: edge.target,
    targetHandle: edge.targetHandle ?? undefined,
  })),
  ...(viewport ? { viewport } : {}),
});

const getAgentflowNodesFromDefinition = (
  definition: AgentflowDefinition | undefined,
) =>
  definition?.nodes.length
    ? definition.nodes.map(node => ({
        id: node.id,
        type: node.type,
        label: node.label,
        x: node.position.x,
        y: node.position.y,
        config: {
          ...createDefaultConfig(node.type),
          ...normalizeDefinitionConfig(node.type, node.config),
        },
      }))
    : [
        {
          id: 'sticky-note-1',
          type: 'sticky-note',
          label: 'Quick Note',
          x: 420,
          y: 160,
          config: {
            note: [
              '# Quick start',
              '',
              '1. Drag Chat Input to define flow arguments.',
              '2. Add Agent Node and write instructions.',
              '3. Add transitions for each route.',
              '4. Connect actions like Transfer Call or End Conversation.',
            ].join('\n'),
            color: 'yellow',
          },
        },
      ];

const getAgentflowEdgesFromDefinition = (
  definition: AgentflowDefinition | undefined,
) =>
  definition?.edges.length
    ? definition.edges.map(edge => ({
        id: edge.id,
        source: edge.source,
        sourceHandle: edge.sourceHandle,
        target: edge.target,
        targetHandle: edge.targetHandle,
        type: 'smoothstep',
        interactionWidth: EDGE_INTERACTION_WIDTH,
        style: EDGE_STYLE,
      }))
    : [];

const getNodeCounterSeed = (nodes: AgentflowNode[]) =>
  Math.max(
    nodes.length + 1,
    ...nodes.map(node => {
      const parts = node.id.split('-');
      const suffix = Number(parts[parts.length - 1]);
      return Number.isFinite(suffix) ? suffix + 1 : 1;
    }),
  );

const createTransitionId = () =>
  `transition-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

const createFallbackTransitionId = (name: string, index: number) => {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '');
  return `transition-${slug || index + 1}`;
};

const createTransitionParameterId = () =>
  `parameter-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

const createFallbackTransitionParameterId = (name: string, index: number) => {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, '-')
    .replace(/(^-|-$)/g, '');
  return `parameter-${slug || index + 1}`;
};

const normalizeFunctionName = (value: string) =>
  value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_]+/g, '_')
    .replace(/^_+|_+$/g, '');

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === 'object' && !Array.isArray(value);

const getParameterTypeFromSchema = (
  schema: Record<string, unknown>,
): string => {
  const anyOf = schema.anyOf;
  if (Array.isArray(anyOf)) {
    const schemaOptions = anyOf.filter(isRecord);
    const nonNullOptions = schemaOptions.filter(
      option => option.type !== 'null',
    );

    if (nonNullOptions.length >= 1) {
      return getParameterTypeFromSchema(nonNullOptions[0]);
    }

    return 'string';
  }

  const type = typeof schema.type === 'string' ? schema.type : '';
  return FUNCTION_PARAMETER_TYPE_VALUES.has(type) ? type : 'string';
};

const createTransitionParameterSchema = (
  parameter: AgentTransitionParameter,
): Record<string, unknown> => {
  const withDescription = (schema: Record<string, unknown>) => ({
    ...schema,
    ...(parameter.description
      ? { description: parameter.description.trim() }
      : {}),
  });

  if (FUNCTION_PARAMETER_TYPE_VALUES.has(parameter.type)) {
    return withDescription({ type: parameter.type });
  }

  return parameter.schema ?? withDescription({ type: 'string' });
};

const getTransitionParameters = (
  transition: Partial<AgentTransition> & {
    properties?: Record<string, unknown>;
    required?: string[];
  },
) => {
  if (Array.isArray(transition.parameters)) {
    return transition.parameters
      .map((parameter, index) => ({
        id:
          parameter.id ||
          createFallbackTransitionParameterId(parameter.name ?? '', index),
        name: String(parameter.name ?? '').trim(),
        type: FUNCTION_PARAMETER_TYPE_VALUES.has(
          String(parameter.type ?? '').trim(),
        )
          ? String(parameter.type ?? '').trim()
          : 'string',
        description: String(parameter.description ?? '').trim(),
        required: Boolean(parameter.required),
        schema: isRecord(parameter.schema) ? parameter.schema : undefined,
      }))
      .filter(parameter => parameter.name || parameter.description);
  }

  const properties = transition.properties;
  if (properties && typeof properties === 'object') {
    const required = Array.isArray(transition.required)
      ? transition.required
      : [];

    return Object.entries(properties).map(([name, schema], index) => {
      const propertySchema =
        schema && typeof schema === 'object'
          ? (schema as Record<string, unknown>)
          : {};

      return {
        id: createFallbackTransitionParameterId(name, index),
        name,
        type: getParameterTypeFromSchema(propertySchema),
        description: String(propertySchema.description ?? ''),
        required: required.includes(name),
        schema: propertySchema,
      };
    });
  }

  return [];
};

const serializeAgentTransitions = (transitions: AgentTransition[]) =>
  JSON.stringify(
    transitions.map(transition => {
      const parameters = getTransitionParameters(transition);
      const properties = parameters.reduce<Record<string, unknown>>(
        (schemaProperties, parameter) => {
          if (!parameter.name) return schemaProperties;
          schemaProperties[parameter.name] =
            createTransitionParameterSchema(parameter);
          return schemaProperties;
        },
        {},
      );

      return {
        id: transition.id,
        name: normalizeFunctionName(transition.name) || transition.name,
        description: transition.description,
        parameters,
        properties,
        required: parameters
          .filter(parameter => parameter.name && parameter.required)
          .map(parameter => parameter.name),
      };
    }),
  );

const createArgumentId = () =>
  `argument-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

const createFallbackArgumentId = (name: string, index: number) => {
  const slug = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '');
  return `argument-${slug || index + 1}`;
};

const serializeChatInputArguments = (argumentsList: ChatInputArgument[]) =>
  JSON.stringify(argumentsList);

const createConditionId = () =>
  `condition-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

const createFallbackConditionId = (field: string, index: number) => {
  const slug = field
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/(^-|-$)/g, '');
  return `condition-${slug || index + 1}`;
};

const serializeConditionRules = (conditions: ConditionRule[]) =>
  JSON.stringify(conditions);

const normalizeConditionOperator = (operator: string) =>
  CONDITION_OPERATOR_OPTIONS.includes(operator) ? operator : 'equals';

const createConditionSourceOptionValue = (
  nodeId: string,
  sourceHandle?: string,
) => [nodeId, sourceHandle].filter(Boolean).join('::');

const getConditionSourceOptionValue = (
  condition: ConditionRule,
  options: ConditionSourceOption[],
) => {
  const exactValue = createConditionSourceOptionValue(
    condition.sourceNodeId,
    condition.sourceHandle,
  );

  if (options.some(option => option.value === exactValue)) {
    return exactValue;
  }

  const fallbackOption = options.find(
    option => option.node.id === condition.sourceNodeId,
  );
  return fallbackOption?.value ?? '';
};

const getConditionRows = (config: AgentflowNodeConfig) => {
  const rawConditions = config.conditions;
  if (typeof rawConditions === 'string' && rawConditions.trim()) {
    try {
      const parsed = JSON.parse(rawConditions) as ConditionRule[];
      if (Array.isArray(parsed)) {
        return parsed
          .filter(condition => condition && typeof condition === 'object')
          .map((condition, index) => ({
            id:
              condition.id ||
              createFallbackConditionId(
                condition.field ?? condition.left ?? '',
                index,
              ),
            sourceNodeId: String(condition.sourceNodeId ?? '').trim(),
            sourceHandle: String(condition.sourceHandle ?? '').trim(),
            field: String(condition.field ?? condition.left ?? '').trim(),
            operator: normalizeConditionOperator(
              String(condition.operator ?? '').trim(),
            ),
            value: String(condition.value ?? '').trim(),
          }));
      }
    } catch {
      return DEFAULT_CONDITION_RULES;
    }
  }

  const legacyValuePath = String(config.value_path ?? '').trim();
  if (legacyValuePath) {
    return [
      {
        id: createFallbackConditionId(legacyValuePath, 0),
        sourceNodeId: '',
        sourceHandle: '',
        field: legacyValuePath,
        operator: normalizeConditionOperator(String(config.operator ?? '')),
        value: String(config.compare_value ?? '').trim(),
      },
    ];
  }

  return DEFAULT_CONDITION_RULES;
};

const getChatInputArgumentRows = (config: AgentflowNodeConfig) => {
  const rawArguments = config.arguments;
  if (typeof rawArguments === 'string' && rawArguments.trim()) {
    try {
      const parsed = JSON.parse(rawArguments) as ChatInputArgument[];
      if (Array.isArray(parsed)) {
        return parsed.map((argument, index) => ({
          id:
            argument.id || createFallbackArgumentId(argument.name ?? '', index),
          name: String(argument.name ?? '').trim(),
          type: String(argument.type ?? 'string').trim() || 'string',
          defaultvalue: String(argument.defaultvalue ?? ''),
        }));
      }
    } catch {
      return [];
    }
  }

  return DEFAULT_CHAT_INPUT_ARGUMENTS;
};

const getAgentTransitionRows = (config: AgentflowNodeConfig) => {
  const rawTransitions = config.transitions;
  if (typeof rawTransitions === 'string' && rawTransitions.trim()) {
    try {
      const parsed = JSON.parse(rawTransitions) as AgentTransition[];
      if (Array.isArray(parsed)) {
        return parsed
          .map((transition, index) => ({
            id:
              transition.id ||
              createFallbackTransitionId(transition.name ?? '', index),
            name: String(transition.name ?? '').trim(),
            description: String(transition.description ?? '').trim(),
            parameters: getTransitionParameters(transition),
          }))
          .filter(
            transition =>
              transition.name ||
              transition.description ||
              transition.parameters.length,
          );
      }
    } catch {
      return rawTransitions
        .split('\n')
        .map(row => row.trim())
        .filter(Boolean)
        .map((row, index) => ({
          id: createFallbackTransitionId(row, index),
          name: normalizeFunctionName(row) || row,
          description: row,
          parameters: [],
        }));
    }
  }

  return DEFAULT_AGENT_TRANSITIONS;
};

const getAgentPromptTemplate = (
  config: AgentflowNodeConfig,
): AgentPromptTemplate => {
  const rawTemplate = config.prompt_template;
  if (typeof rawTemplate === 'string' && rawTemplate.trim()) {
    try {
      const parsed = JSON.parse(rawTemplate) as AgentPromptTemplate;
      if (Array.isArray(parsed.prompt) && Array.isArray(parsed.variables)) {
        return parsed;
      }
    } catch {
      // Ignore invalid saved draft values and fall back to a valid prompt.
    }
  }

  return {
    prompt: [
      {
        role: 'system',
        content: String(config.prompt ?? DEFAULT_AGENT_PROMPT),
      },
    ],
    variables: [],
  };
};

const getAgentPromptPreview = (template: AgentPromptTemplate) =>
  template.prompt.find(message => message.content.trim())?.content ??
  DEFAULT_AGENT_PROMPT;

const serializeMetadataList = (parameters: Metadata[]) =>
  JSON.stringify(
    parameters.map(parameter => ({
      key: parameter.getKey(),
      value: parameter.getValue(),
    })),
  );

const parseMetadataList = (rawValue: AgentflowNodeConfigValue | undefined) => {
  if (typeof rawValue !== 'string' || !rawValue.trim()) {
    return [];
  }

  try {
    const parsed = JSON.parse(rawValue) as { key?: string; value?: string }[];
    if (!Array.isArray(parsed)) {
      return [];
    }

    return parsed
      .filter(item => item.key)
      .map(item => {
        const metadata = new Metadata();
        metadata.setKey(String(item.key));
        metadata.setValue(String(item.value ?? ''));
        return metadata;
      });
  } catch {
    return [];
  }
};

const getAgentModelParameters = (
  config: AgentflowNodeConfig,
  provider: string,
) =>
  GetDefaultTextProviderConfigIfInvalid(
    provider,
    parseMetadataList(config.model_parameters),
  );

const getAgentModelLabel = (config: AgentflowNodeConfig) => {
  const provider = String(config.model_provider ?? DEFAULT_AGENT_PROVIDER);
  const parameters = getAgentModelParameters(config, provider);
  const modelName = parameters
    .find(parameter => parameter.getKey() === 'model.name')
    ?.getValue();
  const modelId = parameters
    .find(parameter => parameter.getKey() === 'model.id')
    ?.getValue();

  return modelName || modelId || provider;
};

const hasAgentCredential = (config: AgentflowNodeConfig) => {
  const provider = String(config.model_provider ?? DEFAULT_AGENT_PROVIDER);
  return getAgentModelParameters(config, provider).some(
    parameter =>
      parameter.getKey() === 'rapida.credential_id' &&
      Boolean(parameter.getValue()),
  );
};

const FUNCTION_IDENTIFIER_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/;

const getNodeValidationIssues = (node: AgentflowNode) => {
  const config = getNodeConfig(node);
  const issues: string[] = [];

  if (node.type === 'sticky-note') {
    return issues;
  }

  if (!node.label.trim()) {
    issues.push('Name is required.');
  }

  getNodeFields(node.type)
    .filter(
      field =>
        field.section === 'basic' &&
        !(
          node.type === 'prompt' &&
          (field.name === 'prompt' || field.name === 'transitions')
        ) &&
        !(node.type === 'chat-input' && field.name === 'arguments'),
    )
    .forEach(field => {
      const value = config[field.name] ?? field.defaultValue;
      if (
        !field.optional &&
        (field.type === 'text' ||
          field.type === 'textarea' ||
          field.type === 'select') &&
        !String(value).trim()
      ) {
        issues.push(`${field.label} is required.`);
      }
    });

  if (node.type === 'prompt') {
    const transitions = getAgentTransitionRows(config);
    const transitionNames = transitions
      .map(transition => transition.name.trim().toLowerCase())
      .filter(Boolean);
    const duplicateTransitionNames = transitionNames.filter(
      (name, index) => transitionNames.indexOf(name) !== index,
    );
    const invalidTransitionNames = transitions.filter(
      transition =>
        transition.name.trim() &&
        !FUNCTION_IDENTIFIER_PATTERN.test(transition.name.trim()),
    );
    const transitionsWithInvalidParameters = transitions.filter(transition =>
      getTransitionParameters(transition).some(
        parameter =>
          !parameter.name.trim() ||
          !FUNCTION_IDENTIFIER_PATTERN.test(parameter.name.trim()) ||
          !parameter.description.trim(),
      ),
    );
    const transitionsWithDuplicateParameters = transitions.filter(
      transition => {
        const parameterNames = getTransitionParameters(transition)
          .map(parameter => parameter.name.trim().toLowerCase())
          .filter(Boolean);
        return parameterNames.some(
          (name, index) => parameterNames.indexOf(name) !== index,
        );
      },
    );

    if (!String(config.prompt ?? '').trim()) {
      issues.push('Instruction is required.');
    }
    if (!hasAgentCredential(config)) {
      issues.push('Credential is required.');
    }
    if (!transitions.length) {
      issues.push('At least one transition is required.');
    }
    if (transitions.some(transition => !transition.name.trim())) {
      issues.push('Every transition needs a name.');
    }
    if (transitions.some(transition => !transition.description.trim())) {
      issues.push('Every transition needs a description.');
    }
    if (duplicateTransitionNames.length) {
      issues.push('Transition names must be unique.');
    }
    if (invalidTransitionNames.length) {
      issues.push(
        'Transition function names must use letters, numbers, or underscores.',
      );
    }
    if (transitionsWithInvalidParameters.length) {
      issues.push('Transition parameters need a valid name and description.');
    }
    if (transitionsWithDuplicateParameters.length) {
      issues.push('Transition parameter names must be unique per transition.');
    }
  }

  if (node.type === 'chat-input') {
    const argumentsList = getChatInputArgumentRows(config);
    const argumentNames = argumentsList
      .map(argument => argument.name.trim().toLowerCase())
      .filter(Boolean);
    const duplicateArgumentNames = argumentNames.filter(
      (name, index) => argumentNames.indexOf(name) !== index,
    );

    if (argumentsList.some(argument => !argument.name.trim())) {
      issues.push('Every argument needs a name.');
    }
    if (argumentsList.some(argument => !argument.type.trim())) {
      issues.push('Every argument needs a type.');
    }
    if (duplicateArgumentNames.length) {
      issues.push('Argument names must be unique.');
    }
  }

  if (node.type === 'condition') {
    const conditions = getConditionRows(config);

    if (conditions.some(condition => !condition.sourceNodeId.trim())) {
      issues.push('Every condition needs a source node.');
    }
    if (conditions.some(condition => !condition.field.trim())) {
      issues.push('Every condition needs an argument or tool parameter.');
    }
    if (conditions.some(condition => !condition.operator.trim())) {
      issues.push('Every condition needs an operator.');
    }
    if (
      conditions.some(
        condition =>
          !['exists', 'is true', 'is false'].includes(condition.operator) &&
          !condition.value.trim(),
      )
    ) {
      issues.push('Every comparison condition needs a value.');
    }
  }

  return issues;
};

const getAgentNodeHeight = (transitionCount: number) =>
  AGENT_NODE_TRANSITION_LIST_TOP +
  Math.max(transitionCount, 1) * AGENT_NODE_TRANSITION_ROW_HEIGHT +
  AGENT_NODE_BOTTOM_PADDING;

const getAgentTransitionPortTop = (index: number) =>
  AGENT_NODE_TRANSITION_LIST_TOP +
  index * AGENT_NODE_TRANSITION_ROW_HEIGHT +
  AGENT_NODE_TRANSITION_ROW_HEIGHT / 2 -
  7;

const getChatInputNodeHeight = (argumentCount: number) =>
  argumentCount === 0
    ? CHAT_INPUT_NODE_HEADER_HEIGHT
    : CHAT_INPUT_NODE_HEADER_HEIGHT +
      CHAT_INPUT_ARGUMENT_HEADER_HEIGHT +
      argumentCount * CHAT_INPUT_ARGUMENT_ROW_HEIGHT +
      CHAT_INPUT_NODE_BOTTOM_PADDING;

const getChatInputOutputPortTop = (argumentCount: number) =>
  getChatInputNodeHeight(argumentCount) / 2 - 7;

const getStaticMessageNodeHeight = (config: AgentflowNodeConfig) =>
  String(config.message ?? '').trim()
    ? COMPACT_NODE_HEADER_HEIGHT + 84
    : COMPACT_NODE_HEADER_HEIGHT;

const getStickyNoteNodeHeight = (config: AgentflowNodeConfig) => {
  const noteLength = String(config.note ?? '').trim().length;
  const lineCount = Math.max(2, Math.ceil(noteLength / 64));

  return Math.min(lineCount, 12) * 40 + 80;
};

const getStickyNoteNodeSize = (node: AgentflowNode) => {
  const config = getNodeConfig(node);
  return {
    width: Number(config.note_width ?? STICKY_NOTE_WIDTH),
    height: Number(config.note_height ?? getStickyNoteNodeHeight(config)),
  };
};

const getStoredNodeStyle = (
  node: AgentflowNode,
): React.CSSProperties | undefined =>
  node.type === 'sticky-note' ? getStickyNoteNodeSize(node) : undefined;

const getConditionNodeHeight = (conditionCount: number) =>
  GENERIC_NODE_HEADER_HEIGHT +
  CONDITION_NODE_CONDITION_HEADER_HEIGHT +
  CONDITION_NODE_ROW_HEIGHT * (conditionCount + 1) +
  CONDITION_NODE_BOTTOM_PADDING;

const getConditionOutputPortTop = (index: number) =>
  GENERIC_NODE_HEADER_HEIGHT +
  CONDITION_NODE_CONDITION_HEADER_HEIGHT +
  index * CONDITION_NODE_ROW_HEIGHT +
  CONDITION_NODE_ROW_HEIGHT / 2;

const getConditionRuleSummary = (condition: ConditionRule) => {
  const left = [condition.sourceNodeId, condition.field]
    .filter(Boolean)
    .join('.');

  if (
    condition.operator === 'exists' ||
    condition.operator === 'is true' ||
    condition.operator === 'is false'
  ) {
    return [left, condition.operator].filter(Boolean).join(' ');
  }

  return [left, condition.operator, condition.value].filter(Boolean).join(' ');
};

const isActionNodeType = (type: AgentflowNodeType) =>
  type === 'end' || type === 'transfer';

const isAnnotationNodeType = (type: AgentflowNodeType) =>
  type === 'sticky-note';

const getNodeOutputs = (node: AgentflowNode) => {
  if (node.type === 'prompt') {
    return getAgentTransitionRows(getNodeConfig(node)).map(
      transition => transition.name,
    );
  }

  if (node.type === 'condition') {
    return [
      ...getConditionRows(getNodeConfig(node)).map(condition => condition.id),
      'else',
    ];
  }

  return getNodeTemplate(node.type).outputs;
};

const getPromptSourceHandleLabel = (
  node: AgentflowNode,
  sourceHandle?: string,
) => {
  if (!sourceHandle) return node.label;

  const transition = getAgentTransitionRows(getNodeConfig(node)).find(
    item => item.id === sourceHandle || item.name === sourceHandle,
  );

  return transition?.name || sourceHandle;
};

const getConditionFieldOptionsForNode = (
  node: AgentflowNode | undefined,
  sourceHandle?: string,
): ConditionFieldOption[] => {
  if (!node) return [];

  const config = getNodeConfig(node);

  if (node.type === 'chat-input') {
    return getChatInputArgumentRows(config)
      .filter(argument => argument.name)
      .map(argument => ({
        value: `argument.${argument.name}`,
        label: `argument.${argument.name}`,
      }));
  }

  if (node.type === 'prompt') {
    const transition = getAgentTransitionRows(config).find(
      item => item.id === sourceHandle || item.name === sourceHandle,
    );

    return transition
      ? getTransitionParameters(transition)
          .filter(parameter => parameter.name)
          .map(parameter => ({
            value: `${transition.name}.${parameter.name}`,
            label: `${transition.name}.${parameter.name}`,
          }))
      : [];
  }

  if (node.type === 'message') {
    return [{ value: 'response', label: 'response' }];
  }

  return getNodeOutputs(node).map(output => ({
    value: output,
    label: output,
  }));
};

const getAgentflowValidationIssues = (
  nodes: AgentflowNode[],
  edges: AgentflowEdge[],
): AgentflowValidationIssue[] => {
  const issues: AgentflowValidationIssue[] = [];
  const nodesById = new Map(nodes.map(node => [node.id, node]));
  const edgeIds = edges.map(edge => edge.id);
  const duplicateEdgeIds = edgeIds.filter(
    (edgeId, index) => edgeIds.indexOf(edgeId) !== index,
  );
  const entryNode = nodesById.get(START_NODE_ID);
  const startNodes = nodes.filter(node => node.type === 'chat-input');

  nodes.forEach(node => {
    getNodeValidationIssues(node).forEach((message, index) => {
      issues.push({
        id: `node-${node.id}-${index}`,
        nodeId: node.id,
        message: `${node.label || node.id}: ${message}`,
      });
    });
  });

  if (!entryNode) {
    issues.push({
      id: 'missing-entry-node',
      message: 'Chat Input entry node is required.',
    });
  }

  if (startNodes.length !== 1) {
    issues.push({
      id: 'invalid-entry-count',
      message: 'Agentflow must have exactly one Chat Input node.',
    });
  }

  if (duplicateEdgeIds.length) {
    issues.push({
      id: 'duplicate-edge-ids',
      message: 'Edge ids must be unique.',
    });
  }

  edges.forEach(edge => {
    if (!nodesById.has(edge.source)) {
      issues.push({
        id: `edge-${edge.id}-source`,
        edgeId: edge.id,
        message: `Edge ${edge.id} has a missing source node.`,
      });
    }
    if (!nodesById.has(edge.target)) {
      issues.push({
        id: `edge-${edge.id}-target`,
        edgeId: edge.id,
        message: `Edge ${edge.id} has a missing target node.`,
      });
    }
  });

  const incomingByNodeId = new Map<string, AgentflowEdge[]>();
  const outgoingByNodeId = new Map<string, AgentflowEdge[]>();
  edges.forEach(edge => {
    incomingByNodeId.set(edge.target, [
      ...(incomingByNodeId.get(edge.target) ?? []),
      edge,
    ]);
    outgoingByNodeId.set(edge.source, [
      ...(outgoingByNodeId.get(edge.source) ?? []),
      edge,
    ]);
  });

  nodes.forEach(node => {
    const template = getNodeTemplate(node.type);
    const incoming = incomingByNodeId.get(node.id) ?? [];
    const outgoing = outgoingByNodeId.get(node.id) ?? [];
    const isTerminal = getNodeOutputs(node).length === 0;

    if (isAnnotationNodeType(node.type)) {
      return;
    }

    if (!isStartNode(node) && template.inputs.length > 0 && !incoming.length) {
      issues.push({
        id: `node-${node.id}-missing-input`,
        nodeId: node.id,
        message: `${node.label || node.id} needs an incoming edge.`,
      });
    }

    if (!isTerminal && !outgoing.length) {
      issues.push({
        id: `node-${node.id}-missing-output`,
        nodeId: node.id,
        message: `${node.label || node.id} needs at least one outgoing edge.`,
      });
    }

    if (node.type === 'condition') {
      getNodeOutputs(node).forEach(output => {
        if (!outgoing.some(edge => edge.sourceHandle === output)) {
          issues.push({
            id: `node-${node.id}-missing-output-${output}`,
            nodeId: node.id,
            message: `${node.label || node.id} output "${output}" needs an edge.`,
          });
        }
      });
    }
  });

  if (entryNode) {
    const visited = new Set<string>();
    const queue = [entryNode.id];

    while (queue.length) {
      const nodeId = queue.shift();
      if (!nodeId || visited.has(nodeId)) continue;
      visited.add(nodeId);
      (outgoingByNodeId.get(nodeId) ?? []).forEach(edge => {
        if (nodesById.has(edge.target)) {
          queue.push(edge.target);
        }
      });
    }

    nodes
      .filter(node => !visited.has(node.id) && !isAnnotationNodeType(node.type))
      .forEach(node => {
        issues.push({
          id: `node-${node.id}-unreachable`,
          nodeId: node.id,
          message: `${node.label || node.id} is not reachable from Chat Input.`,
        });
      });
  }

  return issues;
};

const clamp = (value: number, min: number, max: number) =>
  Math.max(min, Math.min(max, value));

const clampCanvasZoom = (value: number) =>
  clamp(Number(value.toFixed(2)), MIN_CANVAS_ZOOM, MAX_CANVAS_ZOOM);

const getNodeHeight = (type: AgentflowNodeType) => {
  if (type === 'chat-input') {
    return getChatInputNodeHeight(DEFAULT_CHAT_INPUT_ARGUMENTS.length);
  }

  if (type === 'chat-output') {
    return COMPACT_NODE_HEADER_HEIGHT;
  }

  if (type === 'message') {
    return getStaticMessageNodeHeight(createDefaultConfig(type));
  }

  if (type === 'sticky-note') {
    return getStickyNoteNodeHeight(createDefaultConfig(type));
  }

  if (isActionNodeType(type)) {
    return COMPACT_NODE_HEADER_HEIGHT;
  }

  if (type === 'prompt') {
    return getAgentNodeHeight(DEFAULT_AGENT_TRANSITIONS.length);
  }

  if (type === 'condition') {
    return getConditionNodeHeight(DEFAULT_CONDITION_RULES.length);
  }

  const template = getNodeTemplate(type);
  const visibleFields = getNodeFields(type).filter(
    field => field.section === 'basic',
  );
  const visibleFieldCount = Math.min(visibleFields.length, 2);
  const connectorCount = Math.max(
    template.inputs.length,
    template.outputs.length,
    1,
  );

  if (isActionNodeType(type)) {
    return GENERIC_NODE_HEADER_HEIGHT;
  }

  return (
    GENERIC_NODE_HEADER_HEIGHT +
    GENERIC_NODE_SECTION_TOP +
    GENERIC_NODE_SECTION_LABEL_HEIGHT +
    GENERIC_NODE_SECTION_LABEL_BOTTOM_MARGIN +
    visibleFieldCount * GENERIC_NODE_SETTING_ROW_HEIGHT +
    GENERIC_NODE_CONNECTION_SECTION_TOP +
    GENERIC_NODE_CONNECTION_HEADER_HEIGHT +
    connectorCount * GENERIC_NODE_CONNECTION_ROW_HEIGHT +
    GENERIC_NODE_BOTTOM_PADDING
  );
};

const getNodeDisplayHeight = (node: AgentflowNode) => {
  if (node.type === 'chat-input') {
    return getChatInputNodeHeight(
      getChatInputArgumentRows(getNodeConfig(node)).length,
    );
  }

  if (node.type === 'message') {
    return getStaticMessageNodeHeight(getNodeConfig(node));
  }

  if (node.type === 'sticky-note') {
    return getStickyNoteNodeHeight(getNodeConfig(node));
  }

  if (node.type === 'prompt') {
    return getAgentNodeHeight(
      getAgentTransitionRows(getNodeConfig(node)).length,
    );
  }

  if (node.type === 'condition') {
    return getConditionNodeHeight(getConditionRows(getNodeConfig(node)).length);
  }

  return getNodeHeight(node.type);
};

const getNodeAlignmentOffset = (node: AgentflowNode) => {
  if (node.type === 'chat-input') {
    const argumentCount = getChatInputArgumentRows(getNodeConfig(node)).length;
    return getChatInputOutputPortTop(argumentCount) + 7;
  }

  if (node.type === 'prompt') {
    return AGENT_NODE_INPUT_PORT_TOP + 7;
  }

  if (node.type === 'condition') {
    return CONDITION_NODE_INPUT_PORT_TOP + 7;
  }

  return getNodeDisplayHeight(node) / 2;
};

const renderValidationPill = (issues: string[]) => {
  if (!issues.length) return null;

  return (
    <Toggletip align="right">
      <ToggletipButton label="Show validation issues">
        <span className="inline-flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full bg-red-600 px-1 text-[10px] font-semibold leading-none text-white">
          !
        </span>
      </ToggletipButton>
      <ToggletipContent className="normal-case">
        <ul className="m-0 list-disc space-y-1 pl-4">
          {issues.map(issue => (
            <li key={issue}>{issue}</li>
          ))}
        </ul>
      </ToggletipContent>
    </Toggletip>
  );
};

const getNodeCardClass = (selected: boolean, hasValidationError: boolean) =>
  `!min-h-0 !border !p-0 text-left shadow-sm transition dark:!bg-gray-900 ${
    hasValidationError
      ? '!border-red-500 ring-2 ring-red-500/20'
      : selected
        ? '!border-primary ring-2 ring-primary/20'
        : '!border-gray-200 hover:!border-gray-400 dark:!border-gray-800'
  }`;

const getConnectorRingStyle = (colors: string[]): React.CSSProperties => ({
  width: 10,
  height: 10,
  border: 'none',
  zIndex: 20,
  background: colors[0],
});

const AgentflowConnectionLine = ({
  fromX,
  fromY,
  toX,
  toY,
  connectionLineStyle = {},
}: ConnectionLineComponentProps) => (
  <g>
    <path
      fill="none"
      stroke="#0f62fe"
      strokeWidth={1.5}
      d={`M${fromX},${fromY} C ${fromX} ${toY} ${fromX} ${toY} ${toX},${toY}`}
      style={connectionLineStyle}
    />
    <circle
      cx={toX}
      cy={toY}
      r={3}
      fill="#fff"
      stroke="#0f62fe"
      strokeWidth={1.5}
    />
  </g>
);

const AgentflowCanvasNode = ({
  id,
  data,
  isConnectable = true,
  selected = false,
}: NodeProps<AgentflowReactNodeData>) => {
  const {
    node,
    onOpenDetails,
    onSave,
    onDuplicate,
    onDelete,
    onAddTransition,
  } = data;
  const template = getNodeTemplate(node.type);
  const Icon = template.icon;
  const nodeConfig = getNodeConfig(node);
  const nodeOutputs = getNodeOutputs(node);
  const nodeFields = getNodeFields(node.type)
    .filter(field => field.section === 'basic')
    .slice(0, 2);
  const connectorRows = Array.from({
    length: Math.max(template.inputs.length, nodeOutputs.length, 1),
  });
  const transitionRows = getAgentTransitionRows(nodeConfig);
  const conditionRows = getConditionRows(nodeConfig);
  const validationIssues = getNodeValidationIssues(node);
  const hasValidationError = validationIssues.length > 0;
  const cardClass = getNodeCardClass(selected, hasValidationError);
  const updateNodeInternals = useUpdateNodeInternals();
  const isActionGenericNode = isActionNodeType(node.type);
  const isChatOutputNode = (node.type as AgentflowNodeType) === 'chat-output';
  const isAnnotationNode = isAnnotationNodeType(node.type);
  const isCompactGenericNode = isActionGenericNode || isChatOutputNode;
  const getGenericConnectorTop = (index: number) => {
    if (isActionGenericNode || isChatOutputNode) {
      return GENERIC_NODE_HEADER_HEIGHT / 2;
    }

    if (isCompactGenericNode) {
      return (
        GENERIC_NODE_HEADER_HEIGHT +
        index * GENERIC_NODE_CONNECTION_ROW_HEIGHT +
        GENERIC_NODE_CONNECTION_ROW_HEIGHT / 2
      );
    }

    return (
      GENERIC_NODE_HEADER_HEIGHT +
      GENERIC_NODE_SECTION_TOP +
      GENERIC_NODE_SECTION_LABEL_HEIGHT +
      GENERIC_NODE_SECTION_LABEL_BOTTOM_MARGIN +
      nodeFields.length * GENERIC_NODE_SETTING_ROW_HEIGHT +
      GENERIC_NODE_CONNECTION_SECTION_TOP +
      GENERIC_NODE_CONNECTION_HEADER_HEIGHT +
      index * GENERIC_NODE_CONNECTION_ROW_HEIGHT +
      GENERIC_NODE_CONNECTION_ROW_HEIGHT / 2
    );
  };

  useEffect(() => {
    updateNodeInternals(id);
  }, [
    id,
    updateNodeInternals,
    transitionRows.length,
    conditionRows.length,
    nodeOutputs.join('|'),
    template.inputs.join('|'),
  ]);

  const renderSourceHandle = (
    handleId: string,
    style?: React.CSSProperties,
  ) => (
    <Handle
      type="source"
      id={handleId}
      position={Position.Right}
      isConnectable={isConnectable}
      className={`${OUTPUT_CONNECTOR_CLASS} !right-[-5px]`}
      style={{
        ...style,
        ...getConnectorRingStyle(OUTPUT_CONNECTOR_COLORS),
      }}
    />
  );

  const renderTargetHandle = (
    handleId: string,
    style?: React.CSSProperties,
  ) => (
    <Handle
      type="target"
      id={handleId}
      position={Position.Left}
      isConnectable={isConnectable}
      className={`${INPUT_CONNECTOR_CLASS} !left-[-5px]`}
      style={{
        ...style,
        ...getConnectorRingStyle(INPUT_CONNECTOR_COLORS),
      }}
    />
  );

  const toolbar = (
    <NodeToolbar
      isVisible={selected}
      position={Position.Top}
      offset={8}
      className={NODE_TOOLBAR_CLASS}
    >
      {!isAnnotationNode && (
        <Button
          kind="ghost"
          size="sm"
          onClick={() => onOpenDetails(node)}
          className={`${NODE_TOOLBAR_BUTTON_CLASS} nodrag`}
        >
          <Edit size={14} />
          <span>Edit</span>
        </Button>
      )}
      <OverflowMenu
        aria-label="Node options"
        iconDescription="Node options"
        size="sm"
        direction="bottom"
        flipped
        className={`${NODE_TOOLBAR_MENU_CLASS} nodrag`}
      >
        {!isAnnotationNode && (
          <OverflowMenuItem
            itemText="Save"
            className={NODE_TOOLBAR_MENU_ITEM_CLASS}
            onClick={onSave}
          />
        )}
        <OverflowMenuItem
          itemText="Duplicate"
          className={NODE_TOOLBAR_MENU_ITEM_CLASS}
          disabled={isStartNode(node)}
          onClick={() => onDuplicate(node)}
        />
        <OverflowMenuItem
          itemText="Delete"
          isDelete
          className={NODE_TOOLBAR_MENU_ITEM_CLASS}
          disabled={isStartNode(node)}
          onClick={() => onDelete(node)}
        />
      </OverflowMenu>
    </NodeToolbar>
  );

  if (node.type === 'prompt') {
    const agentPromptPreview = String(nodeConfig.prompt ?? '').trim();
    const agentModelLabel = getAgentModelLabel(nodeConfig);

    return (
      <AgentNodeCard
        toolbar={toolbar}
        handles={
          <>
            {renderTargetHandle('input', {
              top: AGENT_NODE_INPUT_PORT_TOP + 7,
            })}
            {transitionRows.map((transition, index) => (
              <React.Fragment key={transition.id}>
                {renderSourceHandle(transition.id, {
                  top: getAgentTransitionPortTop(index) + 7,
                })}
              </React.Fragment>
            ))}
          </>
        }
        Icon={Icon}
        cardClass={cardClass}
        node={node}
        validationPill={renderValidationPill(validationIssues)}
        width={NODE_WIDTH}
        agentModelLabel={agentModelLabel}
        agentPromptPreview={agentPromptPreview}
        transitionRows={transitionRows}
        onAddTransition={event => onAddTransition(event, node)}
      />
    );
  }

  if (node.type === 'chat-input') {
    const inputArguments = getChatInputArgumentRows(nodeConfig);

    return (
      <ChatInputNodeCard
        toolbar={toolbar}
        handles={renderSourceHandle('next', {
          top: getChatInputOutputPortTop(inputArguments.length) + 7,
        })}
        Icon={Icon}
        cardClass={cardClass}
        node={node}
        validationPill={renderValidationPill(validationIssues)}
        width={NODE_WIDTH}
        inputArguments={inputArguments}
      />
    );
  }

  if (node.type === 'message') {
    const messagePreview = String(nodeConfig.message ?? '').trim();
    const postDelayMs = Number(nodeConfig.post_delay_ms ?? 0);

    return (
      <StaticMessageNodeCard
        toolbar={toolbar}
        handles={
          <>
            {renderTargetHandle('incoming')}
            {renderSourceHandle('response')}
          </>
        }
        Icon={Icon}
        cardClass={cardClass}
        node={node}
        validationPill={renderValidationPill(validationIssues)}
        width={NODE_WIDTH}
        messagePreview={messagePreview}
        postDelayMs={postDelayMs}
      />
    );
  }

  if (node.type === 'sticky-note') {
    return (
      <StickyNoteNodeCard
        node={node}
        selected={selected}
        note={String(nodeConfig.note ?? '')}
        noteColor={String(nodeConfig.color ?? 'yellow')}
        onChangeNote={noteValue =>
          data.onUpdateConfig(node.id, 'note', noteValue)
        }
        onResizeEnd={size =>
          data.onUpdateConfigValues(node.id, {
            note_width: Math.round(size.width),
            note_height: Math.round(size.height),
          })
        }
      />
    );
  }

  if (node.type === 'condition') {
    const outputRows = [
      ...conditionRows.map(condition => ({
        id: condition.id,
        label: getConditionRuleSummary(condition) || 'Describe condition',
        prefix: 'If',
      })),
      {
        id: 'else',
        label: 'Else',
        prefix: '',
      },
    ];

    return (
      <ConditionNodeCard
        toolbar={toolbar}
        handles={
          <>
            {renderTargetHandle('value', {
              top: CONDITION_NODE_INPUT_PORT_TOP + 7,
            })}
            {outputRows.map((row, index) => (
              <React.Fragment key={`${node.id}-condition-${row.id}`}>
                {renderSourceHandle(row.id, {
                  top: getConditionOutputPortTop(index),
                })}
              </React.Fragment>
            ))}
          </>
        }
        Icon={Icon}
        cardClass={cardClass}
        node={node}
        validationPill={renderValidationPill(validationIssues)}
        width={NODE_WIDTH}
        outputRows={outputRows}
      />
    );
  }

  return (
    <GenericNodeCard
      toolbar={toolbar}
      handles={
        <>
          {template.inputs.map((input, index) => (
            <React.Fragment key={`${node.id}-input-${input}`}>
              {renderTargetHandle(input, {
                top: getGenericConnectorTop(index),
              })}
            </React.Fragment>
          ))}
          {nodeOutputs.map((output, index) => (
            <React.Fragment key={`${node.id}-output-${output}`}>
              {renderSourceHandle(output, {
                top: getGenericConnectorTop(index),
              })}
            </React.Fragment>
          ))}
        </>
      }
      Icon={Icon}
      cardClass={cardClass}
      node={node}
      validationPill={renderValidationPill(validationIssues)}
      width={NODE_WIDTH}
      connectorRows={connectorRows}
      isCompact={isCompactGenericNode}
      isAction={isActionGenericNode}
      nodeConfig={nodeConfig}
      nodeFields={nodeFields}
      nodeOutputs={nodeOutputs}
      template={template}
    />
  );
};

export type AgentflowVersionBuilderProps = {
  title?: string;
  initialDefinition?: AgentflowDefinition;
  onSave?: (definition: AgentflowDefinition) => void | Promise<void>;
};

export function AgentflowVersionBuilder({
  title = 'Create Agentflow',
  initialDefinition,
  onSave,
}: AgentflowVersionBuilderProps) {
  const { goBack } = useGlobalNavigation();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const canvasRef = useRef<HTMLDivElement | null>(null);
  const edgeUpdateSuccessfulRef = useRef(true);
  const initialCanvasNodes = useMemo(
    () => getAgentflowNodesFromDefinition(initialDefinition),
    [initialDefinition],
  );
  const initialCanvasEdges = useMemo(
    () => getAgentflowEdgesFromDefinition(initialDefinition),
    [initialDefinition],
  );
  const initialSelectedNodeId =
    (initialCanvasNodes[1] ?? initialCanvasNodes[0])?.id ?? null;
  const [storedNodes, setStoredNodes] = useState<AgentflowStoredNode[]>(() =>
    initialCanvasNodes.map(node => ({
      ...createStoredNode(node),
      selected: node.id === initialSelectedNodeId,
    })),
  );
  const [edges, setEdges] = useState<AgentflowEdge[]>(initialCanvasEdges);
  const [selectedNodeId, setSelectedNodeId] = useState(initialSelectedNodeId);
  const [reactFlowInstance, setReactFlowInstance] =
    useState<ReactFlowInstance | null>(null);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const [nodeCounter, setNodeCounter] = useState(
    getNodeCounterSeed(initialCanvasNodes),
  );
  const [componentListOpen, setComponentListOpen] = useState(true);
  const [componentListLocked, setComponentListLocked] = useState(true);
  const [canvasLocked, setCanvasLocked] = useState(false);
  const [canvasZoom, setCanvasZoom] = useState(
    initialDefinition?.viewport?.zoom ?? DEFAULT_CANVAS_ZOOM,
  );
  const [zoomMenuOpen, setZoomMenuOpen] = useState(false);
  const [saveState, setSaveState] = useState<
    'idle' | 'saving' | 'saved' | 'invalid' | 'error'
  >('idle');
  const [saveDialogOpen, setSaveDialogOpen] = useState(false);
  const [exampleDialogOpen, setExampleDialogOpen] = useState(false);
  const [agentflowName, setAgentflowName] = useState(
    initialDefinition?.name ?? '',
  );
  const [agentflowTags, setAgentflowTags] = useState<string[]>(
    initialDefinition?.tags ?? [],
  );
  const [versionMessage, setVersionMessage] = useState(
    `Changed on ${new Date().toLocaleDateString()}`,
  );
  const [selectedExampleName, setSelectedExampleName] = useState<string | null>(
    exampleAgentflowDefinitions[0]?.name ?? null,
  );
  const [activeExampleCategory, setActiveExampleCategory] =
    useState<string>('All');
  const [settingsTransitionId, setSettingsTransitionId] = useState<
    string | null
  >(null);
  const nodes = useMemo(
    () => storedNodes.map(getAgentflowNodeFromStoredNode),
    [storedNodes],
  );
  const validationIssues = useMemo(
    () => getAgentflowValidationIssues(nodes, edges),
    [nodes, edges],
  );

  useEffect(() => {
    if (!initialDefinition) return;

    const nextNodes = getAgentflowNodesFromDefinition(initialDefinition);
    const selectedId = (nextNodes[1] ?? nextNodes[0])?.id ?? null;
    setStoredNodes(
      nextNodes.map(node => ({
        ...createStoredNode(node),
        selected: node.id === selectedId,
      })),
    );
    setEdges(getAgentflowEdgesFromDefinition(initialDefinition));
    setSelectedNodeId(selectedId);
    setNodeCounter(getNodeCounterSeed(nextNodes));
    setCanvasZoom(initialDefinition.viewport?.zoom ?? DEFAULT_CANVAS_ZOOM);
    setAgentflowName(initialDefinition.name ?? '');
    setAgentflowTags(initialDefinition.tags ?? []);
    setDetailsOpen(false);
    setSaveDialogOpen(false);
    setSaveState('idle');
  }, [initialDefinition]);

  const loadAgentflowDefinition = (definition: AgentflowDefinition) => {
    const nextNodes = getAgentflowNodesFromDefinition(definition);
    const selectedId = (nextNodes[1] ?? nextNodes[0])?.id ?? null;

    setStoredNodes(
      nextNodes.map(node => ({
        ...createStoredNode(node),
        selected: node.id === selectedId,
      })),
    );
    setEdges(getAgentflowEdgesFromDefinition(definition));
    setSelectedNodeId(selectedId);
    setNodeCounter(getNodeCounterSeed(nextNodes));
    setCanvasZoom(definition.viewport?.zoom ?? DEFAULT_CANVAS_ZOOM);
    setAgentflowName(definition.name ?? '');
    setAgentflowTags(definition.tags ?? []);
    setDetailsOpen(false);
    setSaveDialogOpen(false);
    setExampleDialogOpen(false);
    setSaveState('idle');
    setZoomMenuOpen(false);

    if (!reactFlowInstance) return;

    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        if (definition.viewport) {
          reactFlowInstance.setViewport(definition.viewport, {
            duration: CANVAS_VIEWPORT_DURATION,
          });
          setCanvasZoom(clampCanvasZoom(definition.viewport.zoom));
          return;
        }

        reactFlowInstance.fitView({
          padding: 0.2,
          duration: CANVAS_VIEWPORT_DURATION,
        });
        closeZoomMenuAndSync(reactFlowInstance);
      });
    });
  };

  const useSelectedExampleDefinition = () => {
    if (!selectedExampleDefinition) return;
    loadAgentflowDefinition(selectedExampleDefinition);
  };

  const setNodes = useCallback(
    (
      update:
        | AgentflowNode[]
        | ((currentNodes: AgentflowNode[]) => AgentflowNode[]),
    ) => {
      setStoredNodes(current => {
        const currentNodes = current.map(getAgentflowNodeFromStoredNode);
        const nextNodes =
          typeof update === 'function' ? update(currentNodes) : update;
        const currentById = new Map(current.map(node => [node.id, node]));

        return nextNodes.map(node => {
          const existing = currentById.get(node.id);

          return {
            ...(existing ?? createStoredNode(node)),
            id: node.id,
            type: 'agentflowNode',
            position: { x: node.x, y: node.y },
            style: getStoredNodeStyle(node),
            data: { node },
            deletable: !isStartNode(node),
          };
        });
      });
    },
    [],
  );

  const selectStoredNode = useCallback((nodeId: string) => {
    setSelectedNodeId(nodeId);
    setStoredNodes(current =>
      current.map(node => ({
        ...node,
        selected: node.id === nodeId,
      })),
    );
  }, []);

  const selectedNode = useMemo(
    () => nodes.find(node => node.id === selectedNodeId) ?? nodes[0],
    [nodes, selectedNodeId],
  );
  const selectedIsAgentNode = selectedNode?.type === 'prompt';
  const selectedIsChatInputNode = selectedNode?.type === 'chat-input';
  const selectedIsChatOutputNode = selectedNode?.type === 'chat-output';
  const selectedIsStaticMessageNode = selectedNode?.type === 'message';
  const selectedIsConditionNode = selectedNode?.type === 'condition';
  const selectedIsActionNode = selectedNode
    ? isActionNodeType(selectedNode.type)
    : false;
  const selectedUsesFocusedDetails =
    selectedIsAgentNode ||
    selectedIsChatInputNode ||
    selectedIsChatOutputNode ||
    selectedIsStaticMessageNode ||
    selectedIsConditionNode ||
    selectedIsActionNode;

  const selectedTemplate = selectedNode
    ? getNodeTemplate(selectedNode.type)
    : null;
  const SelectedIcon = selectedTemplate?.icon;
  const selectedConfig = selectedNode ? getNodeConfig(selectedNode) : {};
  const selectedInputs = selectedTemplate?.inputs ?? [];
  const selectedOutputs = selectedNode ? getNodeOutputs(selectedNode) : [];
  const selectedFields = selectedNode ? getNodeFields(selectedNode.type) : [];
  const selectedBasicFields = selectedFields.filter(
    field =>
      field.section === 'basic' &&
      !(
        selectedIsAgentNode &&
        (field.name === 'prompt' || field.name === 'transitions')
      ) &&
      !(selectedIsChatInputNode && field.name === 'arguments') &&
      !(selectedIsConditionNode && field.name === 'conditions'),
  );
  const selectedTransitionRows = selectedIsAgentNode
    ? getAgentTransitionRows(selectedConfig)
    : [];
  const selectedConditionRows = selectedIsConditionNode
    ? getConditionRows(selectedConfig)
    : [];
  const selectedConditionSourceOptions = selectedIsConditionNode
    ? [
        ...nodes
          .filter(node => node.type === 'chat-input')
          .map(node => ({
            value: createConditionSourceOptionValue(node.id),
            label: node.label || 'Chat Input',
            node,
          })),
        ...edges
          .filter(edge => edge.target === selectedNode?.id)
          .map(edge => {
            const node = nodes.find(item => item.id === edge.source);
            if (!node || node.type === 'chat-input') return null;

            const sourceHandle = edge.sourceHandle ?? undefined;
            return {
              value: createConditionSourceOptionValue(node.id, sourceHandle),
              label:
                node.type === 'prompt'
                  ? `${node.label} / ${getPromptSourceHandleLabel(
                      node,
                      sourceHandle,
                    )}`
                  : node.label,
              node,
              sourceHandle,
            };
          })
          .filter((option): option is ConditionSourceOption => Boolean(option)),
      ].filter(
        (option, index, options) =>
          options.findIndex(item => item.value === option.value) === index,
      )
    : [];
  const settingsTransition =
    selectedTransitionRows.find(
      transition => transition.id === settingsTransitionId,
    ) ?? null;
  const settingsTransitionParameters = settingsTransition
    ? getTransitionParameters(settingsTransition)
    : [];
  const selectedChatInputArguments = selectedIsChatInputNode
    ? getChatInputArgumentRows(selectedConfig)
    : [];
  const selectedAgentProvider = selectedIsAgentNode
    ? String(selectedConfig.model_provider ?? DEFAULT_AGENT_PROVIDER)
    : DEFAULT_AGENT_PROVIDER;
  const selectedAgentModelParameters = selectedIsAgentNode
    ? getAgentModelParameters(selectedConfig, selectedAgentProvider)
    : [];
  const selectedAgentPromptTemplate = selectedIsAgentNode
    ? getAgentPromptTemplate(selectedConfig)
    : { prompt: [], variables: [] };
  const selectedAdvancedFields = selectedFields.filter(
    field => field.section === 'advanced',
  );
  const defaultEdgeOptions = useMemo<DefaultEdgeOptions>(
    () => ({
      type: 'smoothstep',
      interactionWidth: EDGE_INTERACTION_WIDTH,
      style: EDGE_STYLE,
    }),
    [],
  );

  const paletteTemplates = useMemo(
    () =>
      paletteGroups
        .flatMap(group => group.nodeTypes)
        .map(type => nodeTemplates.find(template => template.type === type))
        .filter((template): template is AgentflowNodeTemplate =>
          Boolean(template),
        ),
    [],
  );
  const exampleCategories = useMemo(
    () => [
      'All',
      ...Array.from(
        new Set(
          exampleAgentflowDefinitions.map(
            definition => definition.tags?.[0] ?? 'General',
          ),
        ),
      ),
    ],
    [],
  );
  const visibleExampleDefinitions = useMemo(
    () =>
      activeExampleCategory === 'All'
        ? exampleAgentflowDefinitions
        : exampleAgentflowDefinitions.filter(
            definition =>
              (definition.tags?.[0] ?? 'General') === activeExampleCategory,
          ),
    [activeExampleCategory],
  );
  const selectedExampleDefinition = useMemo(
    () =>
      visibleExampleDefinitions.find(
        definition => definition.name === selectedExampleName,
      ) ??
      exampleAgentflowDefinitions.find(
        definition => definition.name === selectedExampleName,
      ) ??
      visibleExampleDefinitions[0] ??
      null,
    [selectedExampleName, visibleExampleDefinitions],
  );

  useEffect(() => {
    setSaveState(current => (current === 'saved' ? 'idle' : current));
  }, [nodes, edges]);

  const toggleComponentListLock = () => {
    const nextLocked = !componentListLocked;
    setComponentListLocked(nextLocked);
    setComponentListOpen(nextLocked);
  };

  const onComponentListMouseEnter = () => {
    if (!componentListLocked) {
      setComponentListOpen(true);
    }
  };

  const onComponentListMouseLeave = () => {
    if (!componentListLocked) {
      setComponentListOpen(false);
    }
  };

  const syncCanvasZoomFromViewport = (
    instance: ReactFlowInstance | null = reactFlowInstance,
  ) => {
    if (!instance) return;
    setCanvasZoom(clampCanvasZoom(instance.getZoom()));
  };

  const closeZoomMenuAndSync = (
    instance: ReactFlowInstance | null = reactFlowInstance,
  ) => {
    setZoomMenuOpen(false);
    window.setTimeout(() => {
      syncCanvasZoomFromViewport(instance);
    }, CANVAS_VIEWPORT_DURATION);
  };

  const zoomIn = () => {
    if (!reactFlowInstance) return;
    reactFlowInstance.zoomIn({ duration: CANVAS_VIEWPORT_DURATION });
    closeZoomMenuAndSync();
  };

  const zoomOut = () => {
    if (!reactFlowInstance) return;
    reactFlowInstance.zoomOut({ duration: CANVAS_VIEWPORT_DURATION });
    closeZoomMenuAndSync();
  };

  const zoomToActualSize = () => {
    if (!reactFlowInstance) return;
    reactFlowInstance.zoomTo(1, { duration: CANVAS_VIEWPORT_DURATION });
    closeZoomMenuAndSync();
  };

  const zoomToFit = () => {
    if (!reactFlowInstance || nodes.length === 0) return;

    reactFlowInstance.fitView({
      padding: 0.2,
      duration: CANVAS_VIEWPORT_DURATION,
    });
    closeZoomMenuAndSync();
  };

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (!event.metaKey && !event.ctrlKey) return;
      if (!reactFlowInstance) return;

      const key = event.key.toLowerCase();
      if (key === '+' || key === '=') {
        event.preventDefault();
        reactFlowInstance.zoomIn({ duration: CANVAS_VIEWPORT_DURATION });
        closeZoomMenuAndSync(reactFlowInstance);
        return;
      }

      if (key === '-') {
        event.preventDefault();
        reactFlowInstance.zoomOut({ duration: CANVAS_VIEWPORT_DURATION });
        closeZoomMenuAndSync(reactFlowInstance);
        return;
      }

      if (key === '0') {
        event.preventDefault();
        reactFlowInstance.zoomTo(1, { duration: CANVAS_VIEWPORT_DURATION });
        closeZoomMenuAndSync(reactFlowInstance);
        return;
      }

      if (key === '1') {
        event.preventDefault();
        reactFlowInstance.fitView({
          padding: 0.2,
          duration: CANVAS_VIEWPORT_DURATION,
        });
        closeZoomMenuAndSync(reactFlowInstance);
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [closeZoomMenuAndSync, reactFlowInstance]);

  const onPaletteDragStart = (
    event: React.DragEvent,
    template: AgentflowNodeTemplate,
  ) => {
    if (
      template.disabled ||
      (template.type === 'chat-input' && nodes.some(node => isStartNode(node)))
    ) {
      event.preventDefault();
      return;
    }
    event.dataTransfer.effectAllowed = 'copy';
    event.dataTransfer.setData(NODE_DRAG_TYPE, template.type);
  };

  const onReactFlowNodeDragStart = useCallback<NodeDragHandler>(() => {
    setZoomMenuOpen(false);
  }, []);

  const onReactFlowNodeDragStop = useCallback<NodeDragHandler>(
    (_, node) => {
      selectStoredNode(node.id);
    },
    [selectStoredNode],
  );

  const onReactFlowSelectionDragStart =
    useCallback<SelectionDragHandler>(() => {
      setZoomMenuOpen(false);
    }, []);

  const onReactFlowNodesChange = useCallback((changes: NodeChange[]) => {
    const safeChanges = changes.filter(
      change => !(change.type === 'remove' && change.id === START_NODE_ID),
    );
    const removedIds = safeChanges
      .filter(change => change.type === 'remove')
      .map(change => change.id);

    setStoredNodes(current =>
      applyNodeChanges(safeChanges, current).map(node =>
        syncStoredNodeData(node as AgentflowStoredNode),
      ),
    );

    const selectedChange = safeChanges.find(
      change => change.type === 'select' && change.selected,
    );
    if (selectedChange) {
      setSelectedNodeId(selectedChange.id);
    }

    if (removedIds.length) {
      setEdges(current =>
        current.filter(
          edge =>
            !removedIds.includes(edge.source) &&
            !removedIds.includes(edge.target),
        ),
      );
    }
  }, []);

  const onReactFlowEdgesChange = useCallback((changes: EdgeChange[]) => {
    setEdges(current => applyEdgeChanges(changes, current));
  }, []);

  const onReactFlowConnect = useCallback((connection: Connection) => {
    setEdges(current =>
      addEdge(
        {
          ...connection,
          id: `edge-${connection.source}-${connection.sourceHandle}-${connection.target}-${connection.targetHandle}-${Date.now()}`,
          type: 'smoothstep',
          interactionWidth: EDGE_INTERACTION_WIDTH,
          style: EDGE_STYLE,
        },
        current,
      ),
    );
  }, []);

  const onReactFlowEdgeUpdateStart = useCallback(() => {
    edgeUpdateSuccessfulRef.current = false;
  }, []);

  const onReactFlowEdgeUpdate = useCallback(
    (oldEdge: Edge, newConnection: Connection) => {
      edgeUpdateSuccessfulRef.current = true;
      setEdges(current => updateEdge(oldEdge, newConnection, current));
    },
    [],
  );

  const onReactFlowEdgeUpdateEnd = useCallback((_: unknown, edge: Edge) => {
    if (!edgeUpdateSuccessfulRef.current) {
      setEdges(current => current.filter(item => item.id !== edge.id));
    }
    edgeUpdateSuccessfulRef.current = true;
  }, []);

  const onCanvasDrop = (event: React.DragEvent) => {
    event.preventDefault();

    const canvas = canvasRef.current;
    if (!canvas) return;

    const rect = canvas.getBoundingClientRect();
    const dropPosition = reactFlowInstance
      ? reactFlowInstance.project({
          x: event.clientX - rect.left,
          y: event.clientY - rect.top,
        })
      : {
          x: event.clientX - rect.left - NODE_WIDTH / 2,
          y: event.clientY - rect.top - 36,
        };

    const type = event.dataTransfer.getData(
      NODE_DRAG_TYPE,
    ) as AgentflowNodeType;
    if (!type) return;

    if (type === 'chat-input' && nodes.some(node => isStartNode(node))) {
      selectStoredNode(START_NODE_ID);
      setDetailsOpen(false);
      return;
    }

    const template = getNodeTemplate(type);
    const nextNode: AgentflowNode = {
      id: `${type}-${nodeCounter}`,
      type,
      label: template.label,
      x: dropPosition.x,
      y: dropPosition.y,
      config: createDefaultConfig(type),
    };

    setNodes(current => [...current, nextNode]);
    selectStoredNode(nextNode.id);
    setDetailsOpen(type !== 'prompt' && !isAnnotationNodeType(type));
    setNodeCounter(current => current + 1);
  };

  const updateSelectedLabel = (label: string) => {
    if (!selectedNode) return;
    setNodes(current =>
      current.map(node =>
        node.id === selectedNode.id ? { ...node, label } : node,
      ),
    );
  };

  const updateSelectedConfig = (
    name: string,
    value: AgentflowNodeConfigValue,
  ) => {
    if (!selectedNode) return;
    updateNodeConfig(selectedNode.id, name, value);
  };

  const updateNodeConfig = (
    nodeId: string,
    name: string,
    value: AgentflowNodeConfigValue,
  ) => {
    setNodes(current =>
      current.map(node =>
        node.id === nodeId
          ? {
              ...node,
              config: {
                ...getNodeConfig(node),
                [name]: value,
              },
            }
          : node,
      ),
    );
  };

  const updateNodeConfigValues = (
    nodeId: string,
    values: AgentflowNodeConfig,
  ) => {
    setNodes(current =>
      current.map(node =>
        node.id === nodeId
          ? {
              ...node,
              config: {
                ...getNodeConfig(node),
                ...values,
              },
            }
          : node,
      ),
    );
  };

  const updateSelectedTransitions = (transitions: AgentTransition[]) => {
    if (!selectedNode || selectedNode.type !== 'prompt') return;
    updateNodeConfig(
      selectedNode.id,
      'transitions',
      serializeAgentTransitions(transitions),
    );
  };

  const addSelectedTransition = () => {
    const nextTransition: AgentTransition = {
      id: createTransitionId(),
      name: `transition_${selectedTransitionRows.length + 1}`,
      description: '',
      parameters: [],
    };

    updateSelectedTransitions([...selectedTransitionRows, nextTransition]);
  };

  const updateSelectedTransition = (
    id: string,
    values: Partial<AgentTransition>,
  ) => {
    updateSelectedTransitions(
      selectedTransitionRows.map(transition =>
        transition.id === id ? { ...transition, ...values } : transition,
      ),
    );
  };

  const deleteSelectedTransition = (id: string) => {
    updateSelectedTransitions(
      selectedTransitionRows.filter(transition => transition.id !== id),
    );
    setSettingsTransitionId(current => (current === id ? null : current));
  };

  const addSelectedTransitionParameter = (transitionId: string) => {
    updateSelectedTransitions(
      selectedTransitionRows.map(transition => {
        if (transition.id !== transitionId) return transition;

        const parameters = getTransitionParameters(transition);
        return {
          ...transition,
          parameters: [
            ...parameters,
            {
              id: createTransitionParameterId(),
              name: `parameter_${parameters.length + 1}`,
              type: 'string',
              description: '',
              required: true,
            },
          ],
        };
      }),
    );
  };

  const updateSelectedTransitionParameter = (
    transitionId: string,
    parameterId: string,
    values: Partial<AgentTransitionParameter>,
  ) => {
    updateSelectedTransitions(
      selectedTransitionRows.map(transition => {
        if (transition.id !== transitionId) return transition;

        return {
          ...transition,
          parameters: getTransitionParameters(transition).map(parameter =>
            parameter.id === parameterId
              ? { ...parameter, ...values }
              : parameter,
          ),
        };
      }),
    );
  };

  const deleteSelectedTransitionParameter = (
    transitionId: string,
    parameterId: string,
  ) => {
    updateSelectedTransitions(
      selectedTransitionRows.map(transition => {
        if (transition.id !== transitionId) return transition;

        return {
          ...transition,
          parameters: getTransitionParameters(transition).filter(
            parameter => parameter.id !== parameterId,
          ),
        };
      }),
    );
  };

  const updateSelectedChatInputArguments = (
    argumentsList: ChatInputArgument[],
  ) => {
    if (!selectedNode || selectedNode.type !== 'chat-input') return;
    updateNodeConfig(
      selectedNode.id,
      'arguments',
      serializeChatInputArguments(argumentsList),
    );
  };

  const addSelectedChatInputArgument = () => {
    updateSelectedChatInputArguments([
      ...selectedChatInputArguments,
      {
        id: createArgumentId(),
        name: `argument_${selectedChatInputArguments.length + 1}`,
        type: 'string',
        defaultvalue: '',
      },
    ]);
  };

  const updateSelectedChatInputArgument = (
    id: string,
    values: Partial<ChatInputArgument>,
  ) => {
    updateSelectedChatInputArguments(
      selectedChatInputArguments.map(argument =>
        argument.id === id ? { ...argument, ...values } : argument,
      ),
    );
  };

  const deleteSelectedChatInputArgument = (id: string) => {
    updateSelectedChatInputArguments(
      selectedChatInputArguments.filter(argument => argument.id !== id),
    );
  };

  const updateSelectedConditions = (conditions: ConditionRule[]) => {
    if (!selectedNode || selectedNode.type !== 'condition') return;
    updateNodeConfig(
      selectedNode.id,
      'conditions',
      serializeConditionRules(conditions),
    );
  };

  const addSelectedCondition = () => {
    updateSelectedConditions([
      ...selectedConditionRows,
      {
        id: createConditionId(),
        sourceNodeId: '',
        sourceHandle: '',
        field: '',
        operator: 'equals',
        value: '',
      },
    ]);
  };

  const updateSelectedCondition = (
    id: string,
    values: Partial<ConditionRule>,
  ) => {
    updateSelectedConditions(
      selectedConditionRows.map(condition =>
        condition.id === id ? { ...condition, ...values } : condition,
      ),
    );
  };

  const deleteSelectedCondition = (id: string) => {
    updateSelectedConditions(
      selectedConditionRows.filter(condition => condition.id !== id),
    );
  };

  const updateSelectedAgentProvider = (provider: string) => {
    if (!selectedNode || selectedNode.type !== 'prompt') return;
    const parameters = GetDefaultTextProviderConfigOnProviderSwitch(
      provider,
      selectedAgentModelParameters,
    );
    updateNodeConfigValues(selectedNode.id, {
      model_provider: provider,
      model_parameters: serializeMetadataList(parameters),
    });
  };

  const updateSelectedAgentModelParameters = (parameters: Metadata[]) => {
    if (!selectedNode || selectedNode.type !== 'prompt') return;
    updateNodeConfig(
      selectedNode.id,
      'model_parameters',
      serializeMetadataList(parameters),
    );
  };

  const updateSelectedAgentPrompt = (prompt: AgentPromptTemplate) => {
    if (!selectedNode || selectedNode.type !== 'prompt') return;
    updateNodeConfigValues(selectedNode.id, {
      prompt_template: JSON.stringify(prompt),
      prompt: getAgentPromptPreview(prompt),
    });
  };

  const addAgentTransition = (
    event: React.MouseEvent<HTMLButtonElement>,
    node: AgentflowNode,
  ) => {
    event.preventDefault();
    event.stopPropagation();

    setNodes(current =>
      current.map(item => {
        if (item.id !== node.id || item.type !== 'prompt') {
          return item;
        }

        const config = getNodeConfig(item);
        const rows = getAgentTransitionRows(config);
        const nextTransition: AgentTransition = {
          id: createTransitionId(),
          name: `transition_${rows.length + 1}`,
          description: '',
          parameters: [],
        };

        return {
          ...item,
          config: {
            ...config,
            transitions: serializeAgentTransitions([...rows, nextTransition]),
          },
        };
      }),
    );
    selectStoredNode(node.id);
    setDetailsOpen(true);
  };

  const deleteSelectedNode = () => {
    if (!selectedNode || isStartNode(selectedNode)) return;
    const remainingNodes = nodes.filter(node => node.id !== selectedNode.id);
    const fallbackNode =
      remainingNodes.find(node => isStartNode(node)) ?? remainingNodes[0];

    setNodes(remainingNodes);
    setEdges(current =>
      current.filter(
        edge =>
          edge.source !== selectedNode.id && edge.target !== selectedNode.id,
      ),
    );
    setSelectedNodeId(fallbackNode?.id ?? null);
    setDetailsOpen(
      Boolean(fallbackNode && !isAnnotationNodeType(fallbackNode.type)),
    );
  };

  const saveSelectedNode = () => {
    setDetailsOpen(false);
  };

  const getCurrentDefinition = () =>
    createAgentflowDefinition(
      nodes,
      edges,
      reactFlowInstance?.getViewport() ?? {
        x: 0,
        y: 0,
        zoom: canvasZoom,
      },
      {
        name: agentflowName,
        description: versionMessage,
        tags: agentflowTags,
      },
    );

  const focusValidationIssue = (issue: AgentflowValidationIssue) => {
    if (!issue.nodeId) return;
    selectStoredNode(issue.nodeId);
    setDetailsOpen(true);
  };

  const openSaveAgentflowDialog = () => {
    if (validationIssues.length) {
      setSaveState('invalid');
      toast.error(
        `${validationIssues.length} issue${
          validationIssues.length === 1 ? '' : 's'
        } need attention. ${validationIssues[0].message}`,
      );
      focusValidationIssue(validationIssues[0]);
      return;
    }

    setSaveDialogOpen(true);
  };

  const saveAgentflow = async () => {
    const note = versionMessage.trim();
    if (!note) {
      toast.error('Please provide a valid version description.');
      return;
    }

    const definition = getCurrentDefinition();
    setSaveState('saving');

    try {
      if (onSave) {
        await onSave(definition);
      } else if (typeof window !== 'undefined') {
        window.localStorage.setItem(
          'rapida.agentflow.draft',
          JSON.stringify(definition),
        );
      }
      setSaveState('saved');
      setSaveDialogOpen(false);
      toast.success('Agentflow version created successfully.');
    } catch (error) {
      setSaveState('error');
      toast.error(
        error instanceof Error
          ? error.message
          : 'Unable to save agentflow. Please try again later.',
      );
    }
  };

  const applyAutoLayout = () => {
    setNodes(current =>
      current.map((node, index) => ({
        ...node,
        x: FLOW_START_X + index * FLOW_NODE_STEP,
        y: Math.max(
          24,
          Math.round(FLOW_CONNECTOR_BASELINE_Y - getNodeAlignmentOffset(node)),
        ),
      })),
    );
    setZoomMenuOpen(false);

    if (!reactFlowInstance) return;
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        reactFlowInstance.fitView({
          padding: 0.2,
          duration: CANVAS_VIEWPORT_DURATION,
        });
        closeZoomMenuAndSync(reactFlowInstance);
      });
    });
  };

  const openNodeDetails = (node: AgentflowNode) => {
    if (isAnnotationNodeType(node.type)) {
      selectStoredNode(node.id);
      setDetailsOpen(false);
      return;
    }

    selectStoredNode(node.id);
    setDetailsOpen(true);
  };

  const openNodeDetailsFromCanvas = (node: AgentflowNode) => {
    openNodeDetails(node);
  };

  const duplicateNode = (node: AgentflowNode) => {
    if (isStartNode(node)) return;

    const duplicate: AgentflowNode = {
      ...node,
      id: `${node.type}-${nodeCounter}`,
      label: `${node.label} Copy`,
      x: node.x + 32,
      y: node.y + 32,
      config: {
        ...getNodeConfig(node),
      },
    };

    setNodes(current => [...current, duplicate]);
    selectStoredNode(duplicate.id);
    setDetailsOpen(node.type !== 'prompt' && !isAnnotationNodeType(node.type));
    setNodeCounter(current => current + 1);
  };

  const deleteNode = (node: AgentflowNode) => {
    if (isStartNode(node)) return;
    const remainingNodes = nodes.filter(item => item.id !== node.id);
    const fallbackNode =
      remainingNodes.find(item => isStartNode(item)) ?? remainingNodes[0];

    setNodes(remainingNodes);
    setEdges(current =>
      current.filter(
        edge => edge.source !== node.id && edge.target !== node.id,
      ),
    );
    setSelectedNodeId(fallbackNode?.id ?? null);
    setDetailsOpen(
      Boolean(fallbackNode && !isAnnotationNodeType(fallbackNode.type)),
    );
  };

  const renderFieldControl = (field: AgentflowField) => {
    const value = selectedConfig[field.name] ?? field.defaultValue;
    const id = `agentflow-${selectedNode?.id}-${field.name}`;

    if (field.type === 'textarea') {
      return (
        <div>
          <div className="mb-2 flex items-center gap-1">
            <FormLabel htmlFor={id}>{field.label}</FormLabel>
            <Toggletip align="right">
              <ToggletipButton label="Show information">
                <Information size={14} />
              </ToggletipButton>
              <ToggletipContent>{field.description}</ToggletipContent>
            </Toggletip>
          </div>
          <TextArea
            id={id}
            labelText={field.label}
            hideLabel
            value={String(value)}
            onChange={event =>
              updateSelectedConfig(field.name, event.target.value)
            }
            rows={4}
          />
        </div>
      );
    }

    if (field.type === 'select') {
      return (
        <div>
          <div className="mb-2 flex items-center gap-1">
            <FormLabel htmlFor={id}>{field.label}</FormLabel>
            <Toggletip align="right">
              <ToggletipButton label="Show information">
                <Information size={14} />
              </ToggletipButton>
              <ToggletipContent>{field.description}</ToggletipContent>
            </Toggletip>
          </div>
          <Select
            id={id}
            labelText={field.label}
            hideLabel
            value={String(value)}
            onChange={event =>
              updateSelectedConfig(field.name, event.target.value)
            }
            size="md"
          >
            {(field.options ?? []).map(option => (
              <SelectItem key={option} value={option} text={option} />
            ))}
          </Select>
        </div>
      );
    }

    if (field.type === 'toggle') {
      return (
        <div className="border-b border-gray-200 py-3 dark:border-gray-800">
          <div className="mb-2 flex items-center gap-1">
            <FormLabel htmlFor={id}>{field.label}</FormLabel>
            <Toggletip align="right">
              <ToggletipButton label="Show information">
                <Information size={14} />
              </ToggletipButton>
              <ToggletipContent>{field.description}</ToggletipContent>
            </Toggletip>
          </div>
          <Toggle
            id={id}
            labelText={field.label}
            hideLabel
            size="sm"
            labelA="Off"
            labelB="On"
            toggled={Boolean(value)}
            onToggle={checked => updateSelectedConfig(field.name, checked)}
          />
        </div>
      );
    }

    if (field.type === 'number' && field.control === 'slider-input') {
      const numericValue = Number(value);
      const safeValue = Number.isFinite(numericValue)
        ? numericValue
        : Number(field.defaultValue);
      const min = field.min ?? 0;
      const max = field.max ?? 100;
      const step = field.step ?? 1;
      const updateNumberField = (nextValue: number) => {
        updateSelectedConfig(field.name, clamp(Number(nextValue), min, max));
      };

      return (
        <div>
          <div className="mb-2 flex items-center gap-1">
            <FormLabel htmlFor={id}>{field.label}</FormLabel>
            <Toggletip align="right">
              <ToggletipButton label="Show information">
                <Information size={14} />
              </ToggletipButton>
              <ToggletipContent>{field.description}</ToggletipContent>
            </Toggletip>
          </div>
          <div>
            <Slider
              id={`${id}-slider`}
              labelText=""
              min={min}
              max={max}
              step={step}
              value={safeValue}
              onChange={({ value: nextValue }: { value: number }) =>
                updateNumberField(nextValue)
              }
            />
          </div>
        </div>
      );
    }

    return (
      <div>
        <div className="mb-2 flex items-center gap-1">
          <FormLabel htmlFor={id}>{field.label}</FormLabel>
          <Toggletip align="right">
            <ToggletipButton label="Show information">
              <Information size={14} />
            </ToggletipButton>
            <ToggletipContent>{field.description}</ToggletipContent>
          </Toggletip>
        </div>
        <TextInput
          id={id}
          labelText={field.label}
          hideLabel
          value={String(value)}
          type={field.type === 'number' ? 'number' : 'text'}
          onChange={event => {
            const nextValue =
              field.type === 'number'
                ? Number(event.target.value)
                : event.target.value;
            updateSelectedConfig(field.name, nextValue);
          }}
          size="md"
        />
      </div>
    );
  };

  const reactFlowNodes = useMemo<Node<AgentflowReactNodeData>[]>(() => {
    return storedNodes.map(storedNode => {
      const node = getAgentflowNodeFromStoredNode(storedNode);

      return {
        ...storedNode,
        data: {
          node,
          onOpenDetails: openNodeDetailsFromCanvas,
          onSave: saveSelectedNode,
          onDuplicate: duplicateNode,
          onDelete: deleteNode,
          onUpdateConfig: updateNodeConfig,
          onUpdateConfigValues: updateNodeConfigValues,
          onAddTransition: addAgentTransition,
        },
        draggable: !canvasLocked,
        deletable: !isStartNode(node),
      };
    });
  }, [
    storedNodes,
    canvasLocked,
    openNodeDetailsFromCanvas,
    saveSelectedNode,
    duplicateNode,
    deleteNode,
    updateNodeConfig,
    updateNodeConfigValues,
    addAgentTransition,
  ]);

  const reactFlowNodeTypes = useMemo(
    () => ({ agentflowNode: AgentflowCanvasNode }),
    [],
  );

  return (
    <div className="flex h-full flex-col overflow-hidden bg-white dark:bg-gray-950">
      <Helmet title={title} />
      <ConfirmDialogComponent />
      <div
        className={`grid min-h-0 flex-1 border-t border-gray-200 dark:border-gray-800 ${
          detailsOpen
            ? componentListOpen
              ? 'grid-cols-[240px_minmax(0,1fr)_680px]'
              : 'grid-cols-[48px_minmax(0,1fr)_680px]'
            : componentListOpen
              ? 'grid-cols-[240px_minmax(0,1fr)]'
              : 'grid-cols-[48px_minmax(0,1fr)]'
        }`}
      >
        <aside
          onMouseEnter={onComponentListMouseEnter}
          onMouseLeave={onComponentListMouseLeave}
          className="flex min-h-0 flex-col border-r border-gray-200 bg-white transition-all duration-200 dark:border-gray-800 dark:bg-gray-900"
        >
          <div
            className={`flex h-12 shrink-0 items-center border-b border-gray-200 dark:border-gray-800 ${
              componentListOpen ? 'justify-between px-4' : 'justify-center'
            }`}
          >
            {componentListOpen ? (
              <div className="text-sm font-semibold leading-none text-gray-700 dark:text-gray-100">
                Components
              </div>
            ) : (
              <ToolKit size={16} className="text-gray-500 dark:text-gray-400" />
            )}
          </div>
          <div className="min-h-0 flex-1 overflow-auto pb-4">
            {componentListOpen ? (
              <StructuredListWrapper
                aria-label="Agentflow components"
                isCondensed
                isFlush
                className="m-0 !w-full border-0 bg-transparent"
              >
                <StructuredListBody>
                  {paletteTemplates.map(template => {
                    const Icon = template.icon;
                    const templateDisabled =
                      template.disabled ||
                      (template.type === 'chat-input' &&
                        nodes.some(node => isStartNode(node)));
                    return (
                      <StructuredListRow
                        key={template.type}
                        draggable={!templateDisabled}
                        aria-disabled={templateDisabled}
                        onDragStart={event =>
                          onPaletteDragStart(event, template)
                        }
                        className={`!flex !h-10 !w-full !items-center !border-x-0 transition-colors ${
                          templateDisabled
                            ? 'cursor-not-allowed bg-gray-50 text-gray-400 dark:bg-gray-900 dark:text-gray-500'
                            : 'cursor-grab bg-white text-black hover:bg-gray-50 active:cursor-grabbing dark:bg-gray-900 dark:text-white dark:hover:bg-gray-800'
                        }`}
                      >
                        <StructuredListCell className="!flex min-w-0 !flex-1 !items-center border-0 py-0 !pl-4 pr-6">
                          <span className="flex min-w-0 items-center gap-3">
                            <Icon
                              size={16}
                              className={`shrink-0 ${
                                templateDisabled
                                  ? 'text-gray-400 dark:text-gray-500'
                                  : 'text-black dark:text-white'
                              }`}
                            />
                            <span className="truncate text-sm font-normal leading-none">
                              {template.label}
                            </span>
                          </span>
                        </StructuredListCell>
                        <StructuredListCell
                          noWrap
                          className="!flex !w-12 shrink-0 !items-center !justify-center border-0 px-0 py-0"
                        >
                          <Draggable
                            size={16}
                            className="text-gray-500 dark:text-gray-400"
                          />
                        </StructuredListCell>
                      </StructuredListRow>
                    );
                  })}
                </StructuredListBody>
              </StructuredListWrapper>
            ) : (
              <div className="flex flex-col items-center py-2">
                {paletteTemplates.map(template => {
                  const Icon = template.icon;
                  return (
                    <button
                      key={template.type}
                      type="button"
                      onClick={() => setComponentListOpen(true)}
                      className="flex h-10 w-full items-center justify-center text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
                      aria-label={template.label}
                    >
                      <Icon size={16} />
                    </button>
                  );
                })}
              </div>
            )}
          </div>
          <div className="shrink-0 border-t border-gray-200 dark:border-gray-800">
            <button
              type="button"
              onClick={toggleComponentListLock}
              aria-label={
                componentListLocked
                  ? 'Collapse components'
                  : 'Expand components'
              }
              className="flex h-10 w-full cursor-pointer items-center px-4 text-[var(--cds-text-secondary)] transition-colors duration-100 hover:bg-gray-100 hover:text-[var(--cds-text-primary)] dark:hover:bg-gray-800"
            >
              <span className="shrink-0">
                {componentListLocked ? (
                  <SidePanelClose size={16} />
                ) : (
                  <SidePanelOpen size={16} />
                )}
              </span>
              {componentListOpen && (
                <span className="ml-3 truncate text-xs">
                  {componentListLocked ? 'Collapse' : 'Stick open'}
                </span>
              )}
            </button>
          </div>
        </aside>

        <main
          ref={canvasRef}
          className="relative min-h-0 overflow-hidden bg-white dark:bg-gray-950"
        >
          <ReactFlow
            nodes={reactFlowNodes}
            edges={edges}
            nodeTypes={reactFlowNodeTypes}
            onNodesChange={onReactFlowNodesChange}
            onEdgesChange={onReactFlowEdgesChange}
            onConnect={onReactFlowConnect}
            onEdgeUpdate={onReactFlowEdgeUpdate}
            onEdgeUpdateStart={onReactFlowEdgeUpdateStart}
            onEdgeUpdateEnd={onReactFlowEdgeUpdateEnd}
            onNodeDragStart={onReactFlowNodeDragStart}
            onNodeDragStop={onReactFlowNodeDragStop}
            onSelectionDragStart={onReactFlowSelectionDragStart}
            onInit={instance => {
              setReactFlowInstance(instance);
              window.requestAnimationFrame(() => {
                setCanvasZoom(clampCanvasZoom(instance.getZoom()));
              });
            }}
            onDrop={onCanvasDrop}
            onDragOver={event => {
              event.preventDefault();
              event.dataTransfer.dropEffect = 'copy';
            }}
            onNodeClick={(_, reactFlowNode) => {
              selectStoredNode(reactFlowNode.id);
              setZoomMenuOpen(false);
            }}
            onNodeDoubleClick={(_, reactFlowNode) => {
              const node = nodes.find(item => item.id === reactFlowNode.id);
              if (node) {
                openNodeDetails(node);
              }
            }}
            onMoveEnd={(_, viewport) => {
              setCanvasZoom(clampCanvasZoom(viewport.zoom));
            }}
            onPaneClick={() => setZoomMenuOpen(false)}
            defaultViewport={
              initialDefinition?.viewport ?? {
                x: 0,
                y: 0,
                zoom: DEFAULT_CANVAS_ZOOM,
              }
            }
            minZoom={MIN_CANVAS_ZOOM}
            maxZoom={MAX_CANVAS_ZOOM}
            defaultEdgeOptions={defaultEdgeOptions}
            connectionLineComponent={AgentflowConnectionLine}
            panOnDrag={!canvasLocked}
            panActivationKeyCode=""
            zoomOnScroll
            zoomOnPinch
            nodesDraggable={!canvasLocked}
            nodesConnectable
            nodesFocusable={false}
            edgesUpdatable
            edgeUpdaterRadius={16}
            elementsSelectable
            deleteKeyCode={['Backspace', 'Delete']}
            elevateEdgesOnSelect
            elevateNodesOnSelect
            disableKeyboardA11y
            proOptions={{ hideAttribution: true }}
            className="agentflow-react-flow"
          >
            <Background gap={24} size={1} color="rgba(107,114,128,0.25)" />
            <Panel
              position="bottom-center"
              className="nodrag nopan nowheel !m-4 flex h-10 items-center gap-1 border border-gray-200 bg-white px-2 text-gray-500 shadow-sm dark:border-gray-800 dark:bg-gray-900 dark:text-gray-300"
            >
              <div className="relative">
                {zoomMenuOpen && (
                  <div
                    role="menu"
                    className="absolute bottom-11 left-0 w-64 overflow-hidden border border-gray-200 bg-white py-1 text-sm text-gray-700 shadow-lg dark:border-gray-800 dark:bg-gray-900 dark:text-gray-200"
                  >
                    <button
                      type="button"
                      role="menuitem"
                      onClick={zoomIn}
                      className="flex h-10 w-full items-center justify-between gap-4 px-4 text-left hover:bg-gray-100 dark:hover:bg-gray-800"
                    >
                      <span>Zoom In</span>
                      <span className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-800 dark:text-gray-400">
                        Cmd +
                      </span>
                    </button>
                    <button
                      type="button"
                      role="menuitem"
                      onClick={zoomOut}
                      className="flex h-10 w-full items-center justify-between gap-4 px-4 text-left hover:bg-gray-100 dark:hover:bg-gray-800"
                    >
                      <span>Zoom Out</span>
                      <span className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-800 dark:text-gray-400">
                        Cmd -
                      </span>
                    </button>
                    <div className="my-1 border-t border-gray-200 dark:border-gray-800" />
                    <button
                      type="button"
                      role="menuitem"
                      onClick={zoomToActualSize}
                      className="flex h-10 w-full items-center justify-between gap-4 px-4 text-left hover:bg-gray-100 dark:hover:bg-gray-800"
                    >
                      <span>Zoom To 100%</span>
                      <span className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-800 dark:text-gray-400">
                        Cmd 0
                      </span>
                    </button>
                    <button
                      type="button"
                      role="menuitem"
                      onClick={zoomToFit}
                      className="flex h-10 w-full items-center justify-between gap-4 px-4 text-left hover:bg-gray-100 dark:hover:bg-gray-800"
                    >
                      <span>Zoom To Fit</span>
                      <span className="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-gray-800 dark:text-gray-400">
                        Cmd 1
                      </span>
                    </button>
                  </div>
                )}
                <button
                  type="button"
                  aria-label="Canvas zoom"
                  aria-haspopup="menu"
                  aria-expanded={zoomMenuOpen}
                  onClick={() => setZoomMenuOpen(current => !current)}
                  className="flex h-8 min-w-20 items-center justify-center gap-2 px-2 text-sm font-medium text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800"
                >
                  <span>{Math.round(canvasZoom * 100)}%</span>
                  <CaretDown size={16} />
                </button>
              </div>
              <Tooltip
                align="top"
                description={
                  canvasLocked ? 'Unlock node dragging' : 'Lock node dragging'
                }
              >
                <button
                  type="button"
                  aria-label={
                    canvasLocked ? 'Unlock node dragging' : 'Lock node dragging'
                  }
                  onClick={() => setCanvasLocked(current => !current)}
                  className="flex h-8 w-8 items-center justify-center hover:bg-gray-100 dark:hover:bg-gray-800"
                >
                  {canvasLocked ? <Locked size={18} /> : <Unlocked size={18} />}
                </button>
              </Tooltip>
              <Tooltip align="top" description="Auto layout">
                <button
                  type="button"
                  aria-label="Auto layout"
                  onClick={applyAutoLayout}
                  className="flex h-8 w-8 items-center justify-center hover:bg-gray-100 dark:hover:bg-gray-800"
                >
                  <Flow size={18} />
                </button>
              </Tooltip>
              <Tooltip align="top" description="Examples">
                <button
                  type="button"
                  aria-label="Examples"
                  onClick={() => setExampleDialogOpen(true)}
                  className="flex h-8 w-8 items-center justify-center hover:bg-gray-100 dark:hover:bg-gray-800"
                >
                  <Document size={18} />
                </button>
              </Tooltip>
              <div className="mx-1 h-5 border-l border-gray-200 dark:border-gray-800" />
              <Tooltip align="top" description="Cancel">
                <Button
                  kind="danger--ghost"
                  size="sm"
                  hasIconOnly
                  renderIcon={Close}
                  iconDescription="Cancel"
                  onClick={() => showDialog(goBack)}
                  className="!h-8 !min-h-8 !w-8 !p-0 hover:!bg-red-600 hover:!text-white"
                />
              </Tooltip>
              <Tooltip
                align="top"
                description={
                  saveState === 'saving' ? 'Creating version' : 'Create version'
                }
              >
                <Button
                  kind="ghost"
                  size="sm"
                  hasIconOnly
                  renderIcon={Save}
                  iconDescription={
                    saveState === 'saving'
                      ? 'Creating version'
                      : 'Create version'
                  }
                  onClick={openSaveAgentflowDialog}
                  disabled={saveState === 'saving'}
                  className="!h-8 !min-h-8 !w-8 !p-0 !text-primary hover:!bg-primary hover:!text-white disabled:!text-gray-400 disabled:hover:!bg-transparent dark:disabled:!text-gray-600"
                />
              </Tooltip>
            </Panel>
          </ReactFlow>
        </main>

        {detailsOpen && (
          <aside className="flex min-h-0 flex-col border-l border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-900">
            {selectedNode && selectedTemplate ? (
              <>
                <div className="grid grid-cols-[minmax(0,1fr)_auto] border-b border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-950">
                  <div className="flex min-w-0 items-start gap-3 p-4 pr-3">
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-200">
                      {SelectedIcon ? <SelectedIcon size={18} /> : null}
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex min-w-0 items-center gap-2">
                        <div className="truncate text-sm font-semibold text-gray-900 dark:text-white">
                          {selectedNode.label}
                        </div>
                        <Tag type="gray" size="sm">
                          {selectedNode.id}
                        </Tag>
                      </div>
                      <div className="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
                        {selectedTemplate.description}
                      </div>
                    </div>
                  </div>
                  <div className="flex h-12 items-center justify-end p-1 pl-0">
                    <OverflowMenu
                      aria-label="Node options"
                      iconDescription="Node options"
                      size="md"
                      direction="bottom"
                      flipped
                    >
                      <OverflowMenuItem
                        itemText="Delete node"
                        isDelete
                        disabled={isStartNode(selectedNode)}
                        onClick={deleteSelectedNode}
                      />
                    </OverflowMenu>
                    <Button
                      kind="ghost"
                      size="md"
                      hasIconOnly
                      renderIcon={Close}
                      iconDescription="Close details"
                      tooltipPosition="bottom"
                      onClick={() => setDetailsOpen(false)}
                      className="!h-10 !min-h-10 !w-10 !max-w-10 !p-0"
                    />
                  </div>
                </div>

                <div className="overflow-auto">
                  <div className="border-b border-gray-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-950">
                    <div className="space-y-6">
                      <div>
                        <div className="mb-2 flex items-center gap-1">
                          <FormLabel htmlFor="agentflow-selected-node-label">
                            Name
                          </FormLabel>
                          <Toggletip align="right">
                            <ToggletipButton label="Show information">
                              <Information size={14} />
                            </ToggletipButton>
                            <ToggletipContent>
                              Node name shown on the canvas.
                            </ToggletipContent>
                          </Toggletip>
                        </div>
                        <TextInput
                          id="agentflow-selected-node-label"
                          labelText="Name"
                          hideLabel
                          value={selectedNode.label}
                          onChange={event =>
                            updateSelectedLabel(event.target.value)
                          }
                          size="md"
                        />
                      </div>
                      {selectedIsAgentNode && (
                        <>
                          <div className="[&>*+*]:mt-4">
                            <TextProvider
                              provider={selectedAgentProvider}
                              parameters={selectedAgentModelParameters}
                              onChangeProvider={updateSelectedAgentProvider}
                              onChangeParameter={
                                updateSelectedAgentModelParameters
                              }
                            />
                          </div>
                          <div className="[&>fieldset+fieldset]:mt-6">
                            <ConfigPrompt
                              instanceId={`agentflow-${selectedNode.id}`}
                              existingPrompt={selectedAgentPromptTemplate}
                              showRuntimeReplacementHint
                              hideArgumentRuntimeHint
                              enableReservedVariableSuggestions
                              onChange={updateSelectedAgentPrompt}
                            />
                          </div>
                        </>
                      )}
                      {selectedBasicFields.map(field => (
                        <div key={field.name}>{renderFieldControl(field)}</div>
                      ))}
                      {selectedIsChatInputNode && (
                        <div>
                          <div className="mb-3 flex items-center gap-1">
                            <FormLabel>Arguments</FormLabel>
                            <Toggletip align="right">
                              <ToggletipButton label="Show information">
                                <Information size={14} />
                              </ToggletipButton>
                              <ToggletipContent>
                                Arguments available when the workflow begins.
                              </ToggletipContent>
                            </Toggletip>
                          </div>
                          <table className="w-full border-collapse border border-gray-200 text-sm dark:border-gray-700 [&_.cds--form-item]:!m-0 [&_.cds--select-input]:!border-none [&_.cds--select-input]:!outline-none [&_.cds--text-input]:!border-none [&_.cds--text-input]:!outline-none">
                            <thead>
                              <tr className="border-b border-gray-200 bg-gray-50 text-xs font-medium text-gray-500 dark:border-gray-700 dark:bg-gray-900">
                                <th className="w-9 border-r border-gray-200 px-0 py-2 dark:border-gray-700" />
                                <th className="w-[32%] border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                                  Name
                                </th>
                                <th className="w-[24%] border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                                  Type
                                </th>
                                <th className="border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                                  Default value
                                </th>
                                <th className="w-10 px-0 py-2" />
                              </tr>
                            </thead>
                            {selectedChatInputArguments.length > 0 ? (
                              <ReactSortable
                                tag="tbody"
                                list={selectedChatInputArguments}
                                setList={list =>
                                  updateSelectedChatInputArguments(
                                    list.map(item => ({
                                      id: item.id,
                                      name: item.name,
                                      type: item.type,
                                      defaultvalue: item.defaultvalue,
                                    })),
                                  )
                                }
                                handle=".argument-handle"
                                ghostClass="opacity-60"
                                animation={150}
                              >
                                {selectedChatInputArguments.map(
                                  (argument, index) => (
                                    <tr
                                      key={argument.id}
                                      className="border-b border-gray-200 last:border-b-0 dark:border-gray-700"
                                    >
                                      <td className="w-9 border-r border-gray-200 p-0 text-center dark:border-gray-700">
                                        <span className="argument-handle inline-flex cursor-grab items-center justify-center p-2 text-gray-400">
                                          <Draggable size={16} />
                                        </span>
                                      </td>
                                      <td className="border-r border-gray-200 p-0 dark:border-gray-700">
                                        <TextInput
                                          id={`argument-name-${argument.id}`}
                                          labelText={`Argument ${
                                            index + 1
                                          } name`}
                                          hideLabel
                                          value={argument.name}
                                          placeholder="message"
                                          size="md"
                                          onChange={event =>
                                            updateSelectedChatInputArgument(
                                              argument.id,
                                              { name: event.target.value },
                                            )
                                          }
                                        />
                                      </td>
                                      <td className="border-r border-gray-200 p-0 dark:border-gray-700">
                                        <Select
                                          id={`argument-type-${argument.id}`}
                                          labelText={`Argument ${
                                            index + 1
                                          } type`}
                                          hideLabel
                                          value={argument.type}
                                          size="md"
                                          onChange={event =>
                                            updateSelectedChatInputArgument(
                                              argument.id,
                                              { type: event.target.value },
                                            )
                                          }
                                        >
                                          {CHAT_INPUT_ARGUMENT_TYPES.map(
                                            type => (
                                              <SelectItem
                                                key={type}
                                                value={type}
                                                text={type}
                                              />
                                            ),
                                          )}
                                        </Select>
                                      </td>
                                      <td className="border-r border-gray-200 p-0 dark:border-gray-700">
                                        <TextInput
                                          id={`argument-default-${argument.id}`}
                                          labelText={`Argument ${
                                            index + 1
                                          } default value`}
                                          hideLabel
                                          value={argument.defaultvalue}
                                          placeholder="Optional"
                                          size="md"
                                          onChange={event =>
                                            updateSelectedChatInputArgument(
                                              argument.id,
                                              {
                                                defaultvalue:
                                                  event.target.value,
                                              },
                                            )
                                          }
                                        />
                                      </td>
                                      <td className="w-10 p-0 text-center">
                                        <Button
                                          hasIconOnly
                                          renderIcon={TrashCan}
                                          iconDescription="Delete argument"
                                          kind="ghost"
                                          size="sm"
                                          onClick={() =>
                                            deleteSelectedChatInputArgument(
                                              argument.id,
                                            )
                                          }
                                        />
                                      </td>
                                    </tr>
                                  ),
                                )}
                              </ReactSortable>
                            ) : (
                              <tbody>
                                <tr>
                                  <td
                                    colSpan={5}
                                    className="px-4 py-3 text-xs text-gray-500 dark:text-gray-400"
                                  >
                                    No arguments yet. Click{' '}
                                    <strong>Add argument</strong> below.
                                  </td>
                                </tr>
                              </tbody>
                            )}
                          </table>
                          <Button
                            kind="tertiary"
                            size="md"
                            renderIcon={Add}
                            onClick={addSelectedChatInputArgument}
                            className="mt-4 !w-full !max-w-none"
                          >
                            Add argument
                          </Button>
                        </div>
                      )}
                      {selectedIsConditionNode && (
                        <div>
                          <div className="mb-3 flex items-center gap-1">
                            <FormLabel>Conditions</FormLabel>
                            <Toggletip align="right">
                              <ToggletipButton label="Show information">
                                <Information size={14} />
                              </ToggletipButton>
                              <ToggletipContent>
                                Conditions evaluate global Chat Input arguments
                                or tool parameters from the connected Agent
                                transition.
                              </ToggletipContent>
                            </Toggletip>
                          </div>
                          <table className="w-full border-collapse border border-gray-200 text-sm dark:border-gray-700 [&_.cds--form-item]:!m-0 [&_.cds--select-input]:!border-none [&_.cds--select-input]:!outline-none [&_.cds--text-input]:!border-none [&_.cds--text-input]:!outline-none">
                            <thead>
                              <tr className="border-b border-gray-200 bg-gray-50 text-xs font-medium text-gray-500 dark:border-gray-700 dark:bg-gray-900">
                                <th className="w-9 border-r border-gray-200 px-0 py-2 dark:border-gray-700" />
                                <th className="w-[26%] border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                                  Source node
                                </th>
                                <th className="border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                                  Argument / tool parameter
                                </th>
                                <th className="w-[20%] border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                                  Operator
                                </th>
                                <th className="w-[20%] border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                                  Value
                                </th>
                                <th className="w-10 px-0 py-2" />
                              </tr>
                            </thead>
                            {selectedConditionRows.length > 0 ? (
                              <ReactSortable
                                tag="tbody"
                                list={selectedConditionRows}
                                setList={list =>
                                  updateSelectedConditions(
                                    list.map(item => ({
                                      id: item.id,
                                      sourceNodeId: item.sourceNodeId,
                                      sourceHandle: item.sourceHandle,
                                      field: item.field,
                                      operator: item.operator,
                                      value: item.value,
                                    })),
                                  )
                                }
                                handle=".condition-handle"
                                ghostClass="opacity-60"
                                animation={150}
                              >
                                {selectedConditionRows.map(
                                  (condition, index) => {
                                    const sourceValue =
                                      getConditionSourceOptionValue(
                                        condition,
                                        selectedConditionSourceOptions,
                                      );
                                    const sourceOption =
                                      selectedConditionSourceOptions.find(
                                        option => option.value === sourceValue,
                                      ) ?? null;
                                    const fieldOptions =
                                      getConditionFieldOptionsForNode(
                                        sourceOption?.node,
                                        sourceOption?.sourceHandle,
                                      );

                                    return (
                                      <tr
                                        key={condition.id}
                                        className="border-b border-gray-200 last:border-b-0 dark:border-gray-700"
                                      >
                                        <td className="w-9 border-r border-gray-200 p-0 text-center dark:border-gray-700">
                                          <span className="condition-handle inline-flex cursor-grab items-center justify-center p-2 text-gray-400">
                                            <Draggable size={16} />
                                          </span>
                                        </td>
                                        <td className="w-[26%] border-r border-gray-200 p-0 dark:border-gray-700">
                                          <Select
                                            id={`condition-source-${condition.id}`}
                                            labelText={`Condition ${
                                              index + 1
                                            } source node`}
                                            hideLabel
                                            value={sourceValue}
                                            size="md"
                                            onChange={event => {
                                              const nextSourceOption =
                                                selectedConditionSourceOptions.find(
                                                  option =>
                                                    option.value ===
                                                    event.target.value,
                                                );
                                              updateSelectedCondition(
                                                condition.id,
                                                {
                                                  sourceNodeId:
                                                    nextSourceOption?.node.id ??
                                                    '',
                                                  sourceHandle:
                                                    nextSourceOption?.sourceHandle ??
                                                    '',
                                                  field: '',
                                                },
                                              );
                                            }}
                                          >
                                            <SelectItem
                                              value=""
                                              text={
                                                selectedConditionSourceOptions.length
                                                  ? 'Select source node'
                                                  : 'Add Chat Input arguments or connect a node first'
                                              }
                                            />
                                            {selectedConditionSourceOptions.map(
                                              option => (
                                                <SelectItem
                                                  key={option.value}
                                                  value={option.value}
                                                  text={option.label}
                                                />
                                              ),
                                            )}
                                          </Select>
                                        </td>
                                        <td className="border-r border-gray-200 p-0 dark:border-gray-700">
                                          <Select
                                            id={`condition-field-${condition.id}`}
                                            labelText={`Condition ${
                                              index + 1
                                            } field`}
                                            hideLabel
                                            value={condition.field}
                                            size="md"
                                            disabled={!sourceOption}
                                            onChange={event =>
                                              updateSelectedCondition(
                                                condition.id,
                                                { field: event.target.value },
                                              )
                                            }
                                          >
                                            <SelectItem
                                              value=""
                                              text={
                                                sourceOption
                                                  ? 'Select argument or parameter'
                                                  : 'Select source first'
                                              }
                                            />
                                            {fieldOptions.map(option => (
                                              <SelectItem
                                                key={option.value}
                                                value={option.value}
                                                text={option.label}
                                              />
                                            ))}
                                          </Select>
                                        </td>
                                        <td className="w-[20%] border-r border-gray-200 p-0 dark:border-gray-700">
                                          <Select
                                            id={`condition-operator-${condition.id}`}
                                            labelText={`Condition ${
                                              index + 1
                                            } operator`}
                                            hideLabel
                                            value={condition.operator}
                                            size="md"
                                            onChange={event =>
                                              updateSelectedCondition(
                                                condition.id,
                                                {
                                                  operator: event.target.value,
                                                },
                                              )
                                            }
                                          >
                                            {CONDITION_OPERATOR_OPTIONS.map(
                                              operator => (
                                                <SelectItem
                                                  key={operator}
                                                  value={operator}
                                                  text={operator}
                                                />
                                              ),
                                            )}
                                          </Select>
                                        </td>
                                        <td className="w-[20%] border-r border-gray-200 p-0 dark:border-gray-700">
                                          <TextInput
                                            id={`condition-value-${condition.id}`}
                                            labelText={`Condition ${
                                              index + 1
                                            } value`}
                                            hideLabel
                                            value={condition.value}
                                            placeholder="Value"
                                            size="md"
                                            disabled={[
                                              'exists',
                                              'is true',
                                              'is false',
                                            ].includes(condition.operator)}
                                            onChange={event =>
                                              updateSelectedCondition(
                                                condition.id,
                                                { value: event.target.value },
                                              )
                                            }
                                          />
                                        </td>
                                        <td className="w-10 p-0 text-center">
                                          <Button
                                            hasIconOnly
                                            renderIcon={TrashCan}
                                            iconDescription="Delete condition"
                                            kind="ghost"
                                            size="sm"
                                            onClick={() =>
                                              deleteSelectedCondition(
                                                condition.id,
                                              )
                                            }
                                          />
                                        </td>
                                      </tr>
                                    );
                                  },
                                )}
                              </ReactSortable>
                            ) : (
                              <tbody>
                                <tr>
                                  <td
                                    colSpan={6}
                                    className="px-4 py-3 text-xs text-gray-500 dark:text-gray-400"
                                  >
                                    No conditions yet. Click{' '}
                                    <strong>Add condition</strong> below.
                                  </td>
                                </tr>
                              </tbody>
                            )}
                          </table>
                          <Button
                            kind="tertiary"
                            size="md"
                            renderIcon={Add}
                            onClick={addSelectedCondition}
                            className="mt-4 !w-full !max-w-none"
                          >
                            Add condition
                          </Button>
                        </div>
                      )}
                      {selectedIsAgentNode && (
                        <div>
                          <div className="mb-3 flex items-center gap-1">
                            <FormLabel>Transitions</FormLabel>
                            <Toggletip align="right">
                              <ToggletipButton label="Show information">
                                <Information size={14} />
                              </ToggletipButton>
                              <ToggletipContent>
                                Each transition becomes a function schema.
                                Parameters define structured data the LLM should
                                collect before calling it.
                              </ToggletipContent>
                            </Toggletip>
                          </div>
                          <table className="w-full border-collapse border border-gray-200 text-sm dark:border-gray-700 [&_.cds--checkbox-label-text]:hidden [&_.cds--form-item]:!m-0 [&_.cds--list-box]:!border-none [&_.cds--select-input]:!border-none [&_.cds--text-input]:!border-none [&_.cds--text-input]:!outline-none">
                            <thead>
                              <tr className="border-b border-gray-200 bg-gray-50 text-xs font-medium text-gray-500 dark:border-gray-700 dark:bg-gray-900">
                                <th className="w-10 border-r border-gray-200 px-0 py-2 dark:border-gray-700" />
                                <th className="w-[28%] border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                                  Name
                                </th>
                                <th className="border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                                  Description
                                </th>
                                <th className="w-10 border-r border-gray-200 px-0 py-2 dark:border-gray-700">
                                  Setting
                                </th>
                                <th className="w-10 px-0 py-2" />
                              </tr>
                            </thead>
                            {selectedTransitionRows.length > 0 ? (
                              <ReactSortable
                                tag="tbody"
                                list={selectedTransitionRows}
                                setList={list =>
                                  updateSelectedTransitions(
                                    list.map(item => ({
                                      id: item.id,
                                      name: item.name,
                                      description: item.description,
                                      parameters: getTransitionParameters(item),
                                    })),
                                  )
                                }
                                handle=".transition-handle"
                                ghostClass="opacity-60"
                                animation={150}
                              >
                                {selectedTransitionRows.map(transition => {
                                  const parameters =
                                    getTransitionParameters(transition);

                                  return (
                                    <tr
                                      key={transition.id}
                                      className="border-b border-gray-200 dark:border-gray-700"
                                    >
                                      <td className="border-r border-gray-200 p-0 text-center dark:border-gray-700">
                                        <span className="transition-handle inline-flex h-10 w-10 cursor-grab items-center justify-center text-gray-400">
                                          <Draggable size={16} />
                                        </span>
                                      </td>
                                      <td className="border-r border-gray-200 p-0 dark:border-gray-700">
                                        <TextInput
                                          id={`transition-name-${transition.id}`}
                                          labelText="Transition name"
                                          hideLabel
                                          value={transition.name}
                                          placeholder="return_package"
                                          size="md"
                                          onChange={event =>
                                            updateSelectedTransition(
                                              transition.id,
                                              { name: event.target.value },
                                            )
                                          }
                                        />
                                      </td>
                                      <td className="border-r border-gray-200 p-0 dark:border-gray-700">
                                        <TextInput
                                          id={`transition-description-${transition.id}`}
                                          labelText="Transition description"
                                          hideLabel
                                          value={transition.description}
                                          placeholder="Use when the caller wants to return a package."
                                          size="md"
                                          onChange={event =>
                                            updateSelectedTransition(
                                              transition.id,
                                              {
                                                description: event.target.value,
                                              },
                                            )
                                          }
                                        />
                                      </td>
                                      <td className="w-10 border-r border-gray-200 p-0 text-center dark:border-gray-700">
                                        <Button
                                          hasIconOnly
                                          kind="ghost"
                                          size="sm"
                                          renderIcon={SettingsAdjust}
                                          iconDescription={`Configure parameters (${parameters.length})`}
                                          onClick={() =>
                                            setSettingsTransitionId(
                                              transition.id,
                                            )
                                          }
                                        />
                                      </td>
                                      <td className="w-10 p-0 text-center">
                                        <Button
                                          hasIconOnly
                                          renderIcon={TrashCan}
                                          iconDescription="Delete transition"
                                          kind="ghost"
                                          size="sm"
                                          onClick={() =>
                                            deleteSelectedTransition(
                                              transition.id,
                                            )
                                          }
                                        />
                                      </td>
                                    </tr>
                                  );
                                })}
                              </ReactSortable>
                            ) : (
                              <tbody>
                                <tr>
                                  <td
                                    colSpan={5}
                                    className="px-4 py-3 text-xs text-gray-500 dark:text-gray-400"
                                  >
                                    No transitions yet. Click{' '}
                                    <strong>Add transition</strong> below.
                                  </td>
                                </tr>
                              </tbody>
                            )}
                          </table>
                          <Button
                            kind="tertiary"
                            size="md"
                            renderIcon={Add}
                            onClick={addSelectedTransition}
                            className="mt-4 !w-full !max-w-none"
                          >
                            Add transition
                          </Button>
                          <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
                            The connected edge decides the next node after the
                            function call completes.
                          </p>
                        </div>
                      )}
                      {!selectedUsesFocusedDetails &&
                        selectedAdvancedFields.length > 0 && (
                          <div>
                            <div className="mb-3 flex items-center gap-1">
                              <FormLabel>Advanced</FormLabel>
                              <Toggletip align="right">
                                <ToggletipButton label="Show information">
                                  <Information size={14} />
                                </ToggletipButton>
                                <ToggletipContent>
                                  Additional settings for this node.
                                </ToggletipContent>
                              </Toggletip>
                            </div>
                            <div className="space-y-4">
                              {selectedAdvancedFields.map(field => (
                                <div key={field.name}>
                                  {renderFieldControl(field)}
                                </div>
                              ))}
                            </div>
                          </div>
                        )}
                      {!selectedUsesFocusedDetails && (
                        <div>
                          <div className="mb-3 flex items-center gap-1">
                            <FormLabel>Connections</FormLabel>
                            <Toggletip align="right">
                              <ToggletipButton label="Show information">
                                <Information size={14} />
                              </ToggletipButton>
                              <ToggletipContent>
                                Available input and output paths for this node.
                              </ToggletipContent>
                            </Toggletip>
                          </div>
                          <StructuredListWrapper
                            aria-label="Selected node connectors"
                            isCondensed
                            isFlush
                            className="m-0 !w-full"
                          >
                            <StructuredListBody>
                              {(selectedInputs.length > 0
                                ? selectedInputs
                                : ['none']
                              ).map(input => (
                                <StructuredListRow
                                  key={`input-${input}`}
                                  className="!flex !w-full border-b border-gray-200 dark:border-gray-800"
                                >
                                  <StructuredListCell className="!flex !flex-1 items-center gap-2 border-0 px-0 py-2 text-xs text-gray-700 dark:text-gray-300">
                                    <span className="h-2 w-2 rounded-full bg-gray-500" />
                                    <span className="truncate">
                                      Input: {input}
                                    </span>
                                  </StructuredListCell>
                                </StructuredListRow>
                              ))}
                              {(selectedOutputs.length > 0
                                ? selectedOutputs
                                : ['none']
                              ).map(output => (
                                <StructuredListRow
                                  key={`output-${output}`}
                                  className="!flex !w-full border-b border-gray-200 dark:border-gray-800"
                                >
                                  <StructuredListCell className="!flex !flex-1 items-center gap-2 border-0 px-0 py-2 text-xs text-gray-700 dark:text-gray-300">
                                    <span className="h-2 w-2 rounded-full bg-primary" />
                                    <span className="truncate">
                                      Output: {output}
                                    </span>
                                  </StructuredListCell>
                                </StructuredListRow>
                              ))}
                            </StructuredListBody>
                          </StructuredListWrapper>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </>
            ) : null}
          </aside>
        )}
      </div>
      <Modal
        open={exampleDialogOpen}
        onClose={() => setExampleDialogOpen(false)}
        size="lg"
        containerClassName="!h-[90vh] !w-[90vw] !max-h-[90vh] !max-w-[90vw]"
      >
        <ModalHeader
          label="Agentflow"
          title="Select a usecase template"
          onClose={() => setExampleDialogOpen(false)}
        />
        <ModalBody hasScrollingContent>
          <p className="mb-4 text-xs leading-relaxed text-gray-500 dark:text-gray-400">
            Choose a pre-configured agentflow to fill the canvas. You can
            customize every node after selecting.
          </p>

          <div className="mb-4 flex flex-wrap items-center gap-2">
            <ContentSwitcher
              onChange={({ name }) => {
                setActiveExampleCategory(name as string);
                const nextDefinition =
                  name === 'All'
                    ? exampleAgentflowDefinitions[0]
                    : exampleAgentflowDefinitions.find(
                        definition =>
                          (definition.tags?.[0] ?? 'General') === name,
                      );
                setSelectedExampleName(nextDefinition?.name ?? null);
              }}
              selectedIndex={exampleCategories.indexOf(activeExampleCategory)}
              size="sm"
            >
              {exampleCategories.map(category => (
                <Switch key={category} name={category} text={category} />
              ))}
            </ContentSwitcher>
            {selectedExampleDefinition && (
              <span className="ml-auto text-xs text-gray-500 dark:text-gray-400">
                Selected:{' '}
                <span className="font-medium text-gray-900 dark:text-gray-100">
                  {selectedExampleDefinition.name}
                </span>
              </span>
            )}
          </div>

          <div className="grid grid-cols-3 border-l border-t border-gray-200 dark:border-gray-800">
            {visibleExampleDefinitions.map(definition => {
              const isSelected =
                selectedExampleDefinition?.name === definition.name;
              const category = definition.tags?.[0] ?? 'General';
              return (
                <div
                  key={definition.name ?? definition.entryNodeId}
                  role="button"
                  tabIndex={0}
                  onClick={() =>
                    setSelectedExampleName(definition.name ?? null)
                  }
                  onKeyDown={event => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      setSelectedExampleName(definition.name ?? null);
                    }
                  }}
                  className={cn(
                    'relative flex min-h-36 cursor-pointer select-none flex-col border-b border-r border-gray-200 p-4 outline-none transition-colors duration-100 dark:border-gray-800',
                    'focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-primary',
                    isSelected
                      ? 'bg-primary/5 dark:bg-primary/10'
                      : 'hover:bg-gray-100 dark:hover:bg-gray-800',
                  )}
                >
                  <CornerBorderOverlay
                    className={isSelected ? 'opacity-100' : undefined}
                  />

                  <div
                    className={cn(
                      'absolute right-0 top-0 z-20 flex h-6 w-6 items-center justify-center transition-colors duration-100',
                      isSelected ? 'bg-primary' : 'bg-transparent',
                    )}
                  >
                    {isSelected && (
                      <Checkmark size={14} className="text-white" />
                    )}
                  </div>

                  <Tag size="sm" type="blue" className="!mb-2 !self-start">
                    {category}
                  </Tag>

                  <h3 className="mb-1.5 pr-6 text-sm font-semibold leading-snug text-gray-900 dark:text-white">
                    {definition.name}
                  </h3>

                  <p className="mb-4 line-clamp-2 flex-1 text-xs leading-relaxed text-gray-500 dark:text-gray-400">
                    {definition.description}
                  </p>

                  {definition.tags?.length ? (
                    <div className="flex flex-wrap gap-1.5">
                      {definition.tags.slice(1, 4).map(tag => (
                        <Tag
                          key={`${definition.name}-${tag}`}
                          size="sm"
                          type="cool-gray"
                          className="!m-0"
                        >
                          {tag}
                        </Tag>
                      ))}
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        </ModalBody>
        <ModalFooter>
          <SecondaryButton
            size="lg"
            onClick={() => setExampleDialogOpen(false)}
          >
            Close
          </SecondaryButton>
          <PrimaryButton
            size="lg"
            disabled={!selectedExampleDefinition}
            onClick={useSelectedExampleDefinition}
          >
            Use this template
          </PrimaryButton>
        </ModalFooter>
      </Modal>
      <Modal
        open={saveDialogOpen}
        onClose={() => setSaveDialogOpen(false)}
        size="md"
        preventCloseOnClickOutside
      >
        <ModalHeader
          title="Version note"
          onClose={() => setSaveDialogOpen(false)}
        />
        <ModalBody hasForm>
          <div className="space-y-6">
            <div>
              <div className="mb-2 flex items-center gap-1">
                <FormLabel htmlFor="agentflow-version-note">
                  Version note
                </FormLabel>
                <Toggletip align="right">
                  <ToggletipButton label="Show information">
                    <Information size={14} />
                  </ToggletipButton>
                  <ToggletipContent>
                    Briefly describe what changed in this version.
                  </ToggletipContent>
                </Toggletip>
              </div>
              <TextArea
                id="agentflow-version-note"
                labelText="Version note"
                hideLabel
                value={versionMessage}
                onChange={event => setVersionMessage(event.target.value)}
                placeholder="Provide a clear and detailed explanation of the changes made to this agentflow."
                rows={4}
              />
            </div>
          </div>
        </ModalBody>
        <ModalFooter>
          <Button
            kind="secondary"
            size="lg"
            onClick={() => setSaveDialogOpen(false)}
          >
            Cancel
          </Button>
          <Button
            kind="primary"
            size="lg"
            onClick={saveAgentflow}
            disabled={saveState === 'saving'}
          >
            {saveState === 'saving' ? 'Creating...' : 'Create new version'}
          </Button>
        </ModalFooter>
      </Modal>
      <Modal
        open={Boolean(settingsTransition)}
        onClose={() => setSettingsTransitionId(null)}
        size="lg"
      >
        <ModalHeader
          label="Transition"
          title={
            settingsTransition
              ? `${settingsTransition.name || 'Transition'} settings`
              : 'Transition settings'
          }
          onClose={() => setSettingsTransitionId(null)}
        />
        <ModalBody hasForm hasScrollingContent>
          {settingsTransition && (
            <div>
              <div className="mb-6 grid grid-cols-1 gap-4">
                <TextInput
                  id={`transition-settings-name-${settingsTransition.id}`}
                  labelText="Name"
                  value={settingsTransition.name}
                  placeholder="return_package"
                  size="md"
                  onChange={event =>
                    updateSelectedTransition(settingsTransition.id, {
                      name: event.target.value,
                    })
                  }
                />
                <TextInput
                  id={`transition-settings-description-${settingsTransition.id}`}
                  labelText="Description"
                  value={settingsTransition.description}
                  placeholder="Use when the caller wants to return a package."
                  size="md"
                  onChange={event =>
                    updateSelectedTransition(settingsTransition.id, {
                      description: event.target.value,
                    })
                  }
                />
              </div>
              <div className="mb-3 flex items-center gap-1">
                <FormLabel>Parameters</FormLabel>
                <Toggletip align="right">
                  <ToggletipButton label="Show information">
                    <Information size={14} />
                  </ToggletipButton>
                  <ToggletipContent>
                    Parameters define structured data the LLM should collect
                    before calling this transition function.
                  </ToggletipContent>
                </Toggletip>
              </div>
              <table className="w-full border-collapse border border-gray-200 text-sm dark:border-gray-700 [&_.cds--checkbox-label-text]:hidden [&_.cds--form-item]:!m-0 [&_.cds--list-box]:!border-none [&_.cds--select-input]:!border-none [&_.cds--text-input]:!border-none [&_.cds--text-input]:!outline-none">
                <thead>
                  <tr className="border-b border-gray-200 bg-gray-50 text-xs font-medium text-gray-500 dark:border-gray-700 dark:bg-gray-900">
                    <th className="w-[24%] border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                      Name
                    </th>
                    <th className="w-[18%] border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                      Type
                    </th>
                    <th className="border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                      Description
                    </th>
                    <th className="w-24 border-r border-gray-200 px-3 py-2 text-left dark:border-gray-700">
                      Required
                    </th>
                    <th className="w-10 px-0 py-2" />
                  </tr>
                </thead>
                {settingsTransitionParameters.length > 0 ? (
                  <tbody>
                    {settingsTransitionParameters.map(parameter => (
                      <tr
                        key={parameter.id}
                        className="border-b border-gray-200 last:border-b-0 dark:border-gray-700"
                      >
                        <td className="border-r border-gray-200 p-0 dark:border-gray-700">
                          <TextInput
                            id={`transition-settings-parameter-name-${parameter.id}`}
                            labelText="Parameter name"
                            hideLabel
                            value={parameter.name}
                            placeholder="order_id"
                            size="md"
                            onChange={event =>
                              updateSelectedTransitionParameter(
                                settingsTransition.id,
                                parameter.id,
                                {
                                  name: event.target.value,
                                },
                              )
                            }
                          />
                        </td>
                        <td className="border-r border-gray-200 p-0 dark:border-gray-700">
                          <Select
                            id={`transition-settings-parameter-type-${parameter.id}`}
                            labelText="Parameter type"
                            hideLabel
                            size="md"
                            value={parameter.type}
                            onChange={event =>
                              updateSelectedTransitionParameter(
                                settingsTransition.id,
                                parameter.id,
                                {
                                  type: event.target.value,
                                },
                              )
                            }
                          >
                            {FUNCTION_PARAMETER_TYPE_OPTIONS.map(option => (
                              <SelectItem
                                key={option.value}
                                value={option.value}
                                text={option.label}
                              />
                            ))}
                          </Select>
                        </td>
                        <td className="border-r border-gray-200 p-0 dark:border-gray-700">
                          <TextInput
                            id={`transition-settings-parameter-description-${parameter.id}`}
                            labelText="Parameter description"
                            hideLabel
                            value={parameter.description}
                            placeholder="The caller's order id."
                            size="md"
                            onChange={event =>
                              updateSelectedTransitionParameter(
                                settingsTransition.id,
                                parameter.id,
                                {
                                  description: event.target.value,
                                },
                              )
                            }
                          />
                        </td>
                        <td className="border-r border-gray-200 px-3 py-0 dark:border-gray-700">
                          <Checkbox
                            id={`transition-settings-parameter-required-${parameter.id}`}
                            labelText=""
                            checked={parameter.required}
                            onChange={() =>
                              updateSelectedTransitionParameter(
                                settingsTransition.id,
                                parameter.id,
                                {
                                  required: !parameter.required,
                                },
                              )
                            }
                          />
                        </td>
                        <td className="w-10 p-0 text-center">
                          <Button
                            hasIconOnly
                            renderIcon={TrashCan}
                            iconDescription="Delete parameter"
                            kind="ghost"
                            size="sm"
                            onClick={() =>
                              deleteSelectedTransitionParameter(
                                settingsTransition.id,
                                parameter.id,
                              )
                            }
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                ) : (
                  <tbody>
                    <tr>
                      <td
                        colSpan={5}
                        className="px-4 py-3 text-xs text-gray-500 dark:text-gray-400"
                      >
                        No parameters yet. Click <strong>Add parameter</strong>{' '}
                        below.
                      </td>
                    </tr>
                  </tbody>
                )}
              </table>
              <Button
                kind="tertiary"
                size="md"
                renderIcon={Add}
                onClick={() =>
                  addSelectedTransitionParameter(settingsTransition.id)
                }
                className="mt-4 !w-full !max-w-none"
              >
                Add parameter
              </Button>
            </div>
          )}
        </ModalBody>
        <ModalFooter>
          <Button
            kind="primary"
            size="lg"
            onClick={() => setSettingsTransitionId(null)}
          >
            Save
          </Button>
        </ModalFooter>
      </Modal>
    </div>
  );
}
