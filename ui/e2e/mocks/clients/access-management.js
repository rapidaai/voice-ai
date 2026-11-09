const { Timestamp } = require('google-protobuf/google/protobuf/timestamp_pb');
const { fulfillGrpcWeb } = require('../grpc-web');

const previousWindow = global.window;
const previousSelf = global.self;
global.window = global;
global.self = global;
const {
  AuditActor,
  ArchiveCustomRoleRequest,
  ArchiveCustomRoleResponse,
  CreateCustomRoleResponse,
  GetRoleRequest,
  GetRoleResponse,
  ListPermissionsResponse,
  ListRolesResponse,
  Paginated,
  PermissionDefinition,
  RoleDefinition,
  RoleDefinitionType,
  RoleScope,
  RoleSubject,
  UpdateCustomRoleRequest,
  UpdateCustomRoleResponse,
} = require('@rapidaai/react');
if (previousWindow === undefined) delete global.window;
else global.window = previousWindow;
if (previousSelf === undefined) delete global.self;
else global.self = previousSelf;

const ACCESS_MANAGEMENT_PATHS = Object.freeze({
  archiveRole: '**/web_api.AccessManagementService/ArchiveCustomRole',
  createRole: '**/web_api.AccessManagementService/CreateCustomRole',
  getRole: '**/web_api.AccessManagementService/GetRole',
  listPermissions: '**/web_api.AccessManagementService/ListPermissions',
  listRoles: '**/web_api.AccessManagementService/ListRoles',
  updateRole: '**/web_api.AccessManagementService/UpdateCustomRole',
});

function timestamp(seconds) {
  const value = new Timestamp();
  value.setSeconds(seconds);
  return value;
}

function auditActor(id, name) {
  const actor = new AuditActor();
  actor.setType('user');
  actor.setId(id);
  actor.setDisplayname(name);
  return actor;
}

function permission(
  id,
  key,
  name,
  description,
  scope = RoleScope.ROLE_SCOPE_PROJECT,
) {
  const value = new PermissionDefinition();
  value.setId(id);
  value.setKey(key);
  value.setScope(scope);
  value.setName(name);
  value.setDescription(description);
  return value;
}

const PROJECT_READ = permission(
  '34',
  'project.read',
  'Project Read',
  'Read project configuration and metadata.',
);
const PROJECT_UPDATE = permission(
  '35',
  'project.update',
  'Project Update',
  'Update project configuration and metadata.',
);

function role({
  id,
  key,
  name,
  description,
  definitionType,
  scope = RoleScope.ROLE_SCOPE_PROJECT,
  subject = RoleSubject.ROLE_SUBJECT_USER,
  permissions = [PROJECT_READ, PROJECT_UPDATE],
}) {
  const value = new RoleDefinition();
  value.setId(id);
  value.setKey(key);
  value.setName(name);
  value.setDescription(description);
  value.setScope(scope);
  value.setSubject(subject);
  value.setDefinitiontype(definitionType);
  value.setCreateddate(timestamp(1704067200));
  value.setUpdateddate(timestamp(1735689600));
  value.setCreatedactor(auditActor('101', 'Ada Lovelace'));
  value.setUpdatedactor(auditActor('102', 'Grace Hopper'));
  value.setPermissionsList(permissions);
  return value;
}

const CUSTOM_ROLE = role({
  id: '501',
  key: 'support_manager',
  name: 'Support manager',
  description: 'Manages support workflows across assigned projects.',
  definitionType: RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM,
});
const BUILT_IN_ROLE = role({
  id: '502',
  key: 'project_viewer',
  name: 'Project viewer',
  description: 'Built-in read-only project access.',
  definitionType: RoleDefinitionType.ROLE_DEFINITION_TYPE_BUILT_IN,
  permissions: [PROJECT_READ],
});
const CREATED_ROLE = role({
  id: '503',
  key: 'incident_manager',
  name: 'Incident manager',
  description: 'Coordinates incident response.',
  definitionType: RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM,
  permissions: [PROJECT_READ],
});

function paginated(total) {
  const value = new Paginated();
  value.setCurrentpage(1);
  value.setTotalitem(total);
  return value;
}

function buildRoleListResponse(roles = [CUSTOM_ROLE, BUILT_IN_ROLE]) {
  const response = new ListRolesResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setRolesList(roles);
  response.setPaginated(paginated(roles.length));
  return Buffer.from(response.serializeBinary());
}

function buildRoleDetailResponse(currentRole = CUSTOM_ROLE) {
  const response = new GetRoleResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setRole(currentRole);
  return Buffer.from(response.serializeBinary());
}

function buildPermissionListResponse() {
  const response = new ListPermissionsResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setPermissionsList([PROJECT_READ, PROJECT_UPDATE]);
  response.setPaginated(paginated(2));
  return Buffer.from(response.serializeBinary());
}

function buildCreateRoleResponse() {
  const response = new CreateCustomRoleResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setRole(CREATED_ROLE);
  return Buffer.from(response.serializeBinary());
}

function buildUpdateRoleResponse(currentRole) {
  const response = new UpdateCustomRoleResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setRole(currentRole);
  return Buffer.from(response.serializeBinary());
}

function buildArchiveRoleResponse(roleDefinitionId) {
  const response = new ArchiveCustomRoleResponse();
  response.setCode(200);
  response.setSuccess(true);
  response.setRoledefinitionid(roleDefinitionId);
  return Buffer.from(response.serializeBinary());
}

function deserializeRequest(route, RequestType) {
  const body = route.request().postDataBuffer();
  const messageLength = body?.readUInt32BE(1) || 0;
  return RequestType.deserializeBinary(
    body?.subarray(5, 5 + messageLength) || Buffer.alloc(0),
  );
}

function fulfillGrpcWebError(route, code, message) {
  const trailers = Buffer.from(
    `grpc-status: ${code}\r\ngrpc-message: ${encodeURIComponent(message)}\r\n`,
  );
  const trailerHeader = Buffer.alloc(5);
  trailerHeader[0] = 0x80;
  trailerHeader.writeUInt32BE(trailers.length, 1);
  return route.fulfill({
    status: 200,
    headers: {
      'access-control-allow-origin': '*',
      'access-control-expose-headers': 'grpc-status, grpc-message',
      'content-type': 'application/grpc-web+proto',
    },
    body: Buffer.concat([trailerHeader, trailers]),
  });
}

async function mockRoleNotFound(page) {
  await page.route(ACCESS_MANAGEMENT_PATHS.getRole, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    return fulfillGrpcWebError(route, 5, 'role not found');
  });
}

async function mockRoleListFailureAfterInitialLoad(page) {
  let requestCount = 0;
  await page.route(ACCESS_MANAGEMENT_PATHS.listRoles, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    requestCount += 1;
    if (requestCount === 1) return route.fallback();
    return fulfillGrpcWebError(route, 13, 'catalog refresh unavailable');
  });
}

async function installAccessManagementClientMock(page) {
  let currentCustomRole = role({
    id: CUSTOM_ROLE.getId(),
    key: CUSTOM_ROLE.getKey(),
    name: CUSTOM_ROLE.getName(),
    description: CUSTOM_ROLE.getDescription(),
    definitionType: RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM,
  });
  let isCustomRoleArchived = false;
  const responses = new Map([
    [ACCESS_MANAGEMENT_PATHS.listPermissions, buildPermissionListResponse()],
    [ACCESS_MANAGEMENT_PATHS.createRole, buildCreateRoleResponse()],
  ]);

  for (const [path, response] of responses) {
    await page.route(path, route => {
      if (route.request().method() !== 'POST') return route.fallback();
      return fulfillGrpcWeb(route, response);
    });
  }

  await page.route(ACCESS_MANAGEMENT_PATHS.listRoles, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    const customRoleSummary = role({
      id: currentCustomRole.getId(),
      key: currentCustomRole.getKey(),
      name: currentCustomRole.getName(),
      description: currentCustomRole.getDescription(),
      definitionType: RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM,
      permissions: [],
    });
    const roles = isCustomRoleArchived
      ? [BUILT_IN_ROLE]
      : [customRoleSummary, BUILT_IN_ROLE];
    return fulfillGrpcWeb(route, buildRoleListResponse(roles));
  });

  await page.route(ACCESS_MANAGEMENT_PATHS.getRole, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    const request = deserializeRequest(route, GetRoleRequest);
    const currentRole =
      request.getRoledefinitionid() === CREATED_ROLE.getId()
        ? CREATED_ROLE
        : request.getRoledefinitionid() === BUILT_IN_ROLE.getId()
          ? BUILT_IN_ROLE
          : currentCustomRole;
    return fulfillGrpcWeb(route, buildRoleDetailResponse(currentRole));
  });

  await page.route(ACCESS_MANAGEMENT_PATHS.updateRole, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    const request = deserializeRequest(route, UpdateCustomRoleRequest);
    currentCustomRole.setName(request.getName());
    currentCustomRole.setDescription(request.getDescription());
    return fulfillGrpcWeb(route, buildUpdateRoleResponse(currentCustomRole));
  });

  await page.route(ACCESS_MANAGEMENT_PATHS.archiveRole, route => {
    if (route.request().method() !== 'POST') return route.fallback();
    const request = deserializeRequest(route, ArchiveCustomRoleRequest);
    if (!request.getReason()) {
      return fulfillGrpcWebError(route, 3, 'deletion reason is required');
    }
    isCustomRoleArchived = true;
    return fulfillGrpcWeb(
      route,
      buildArchiveRoleResponse(request.getRoledefinitionid()),
    );
  });
}

module.exports = {
  ACCESS_MANAGEMENT_PATHS,
  buildCreateRoleResponse,
  buildPermissionListResponse,
  buildRoleDetailResponse,
  buildRoleListResponse,
  installAccessManagementClientMock,
  mockRoleListFailureAfterInitialLoad,
  mockRoleNotFound,
};
