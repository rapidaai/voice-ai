require("../../react-environment");

const {
  AddUserToProjects,
  AddUserToProjectsRequest,
  ConnectionConfig,
  CreateProject,
  GetProject,
  ProjectRoleAssignment,
  UpdateProject,
} = require("@rapidaai/react");

const webEndpoint = process.env.WEB_API_REACT_ENDPOINT || "http://web-api:9001";

function requireValue(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

function call(invoke) {
  return new Promise((resolve, reject) => {
    invoke((error, response) => {
      if (error) {
        reject(error);
        return;
      }
      resolve(response);
    });
  });
}

function requireSuccess(response, action) {
  if (!response?.getSuccess() || response.getCode() !== 200) {
    const message = response?.getError()?.getHumanmessage() || "unknown error";
    throw new Error(
      `${action} failed with code ${response?.getCode()}: ${message}`,
    );
  }
}

function assertProject(project, projectID, name, description) {
  if (
    !project ||
    project.getId() !== projectID ||
    project.getName() !== name ||
    project.getDescription() !== description ||
    project.getStatus() !== "ACTIVE"
  ) {
    throw new Error("project lookup returned unexpected data");
  }
}

async function getProject(connection, projectID, auth, action) {
  const response = await call((callback) =>
    GetProject(connection, projectID, callback, auth),
  );
  requireSuccess(response, action);
  return response.getData();
}

async function main() {
  const connection = new ConnectionConfig(
    { web: webEndpoint },
    process.env.FLOW_SDK_DEBUG === "true",
  );
  const auth = {
    authorization: requireValue("FLOW_TOKEN"),
    "x-auth-id": requireValue("FLOW_OWNER_ID"),
  };
  const projectName = requireValue("FLOW_PROJECT");
  const initialDescription = "Created by the CI React project lifecycle flow";

  const createResponse = await call((callback) =>
    CreateProject(
      connection,
      projectName,
      initialDescription,
      auth,
      callback,
    ),
  );
  requireSuccess(createResponse, "project creation");
  const projectID = createResponse.getData()?.getId();
  if (!projectID) {
    throw new Error("project creation returned no identifier");
  }
  assertProject(
    createResponse.getData(),
    projectID,
    projectName,
    initialDescription,
  );

  const createdProject = await getProject(
    connection,
    projectID,
    auth,
    "created project lookup",
  );
  assertProject(createdProject, projectID, projectName, initialDescription);

  const updatedName = requireValue("FLOW_UPDATED_PROJECT");
  const updatedDescription = "Updated by the CI React project lifecycle flow";
  const updateResponse = await call((callback) =>
    UpdateProject(
      connection,
      projectID,
      callback,
      auth,
      updatedName,
      updatedDescription,
    ),
  );
  requireSuccess(updateResponse, "project update");

  const updatedProject = await getProject(
    connection,
    projectID,
    auth,
    "updated project lookup",
  );
  assertProject(updatedProject, projectID, updatedName, updatedDescription);

  const assignment = new ProjectRoleAssignment();
  assignment.setProjectid(projectID);
  assignment.setProjectrole("writer");
  const addUserRequest = new AddUserToProjectsRequest();
  addUserRequest.setUserid(requireValue("FLOW_MEMBER_ID"));
  addUserRequest.addProjectroles(assignment);
  const addUserResponse = await AddUserToProjects(
    connection,
    addUserRequest,
    auth,
  );
  requireSuccess(addUserResponse, "add user to project");
  if (!addUserResponse.getDataList().some((project) => project.getId() === projectID)) {
    throw new Error("add user response did not include the project");
  }

  const projectWithMember = await getProject(
    connection,
    projectID,
    auth,
    "project membership lookup",
  );
  assertProject(projectWithMember, projectID, updatedName, updatedDescription);
  const members = projectWithMember.getMembersList();
  const owner = members.find(
    (member) => member.getId() === requireValue("FLOW_OWNER_ID"),
  );
  const addedMember = members.find(
    (member) => member.getId() === requireValue("FLOW_MEMBER_ID"),
  );
  if (
    members.length !== 2 ||
    owner?.getRole() !== "admin" ||
    addedMember?.getRole() !== "writer" ||
    addedMember.getEmail() !== requireValue("FLOW_MEMBER_EMAIL")
  ) {
    throw new Error("project membership lookup returned unexpected users");
  }

  console.log("React SDK project lifecycle flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);
