const {
  AuthenticateUser,
  ConnectionConfig,
  CreateOrganization,
  CreateProject,
  RegisterUser,
} = require("@rapidaai/nodejs");

const webEndpoint = process.env.WEB_API_ENDPOINT || "web-api:9001";

function requireValue(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

function requireSuccess(response, action) {
  if (!response.getSuccess() || response.getCode() !== 200) {
    const message = response.getError()?.getHumanmessage() || "unknown error";
    throw new Error(`${action} failed with code ${response.getCode()}: ${message}`);
  }
}

async function main() {
  const email = requireValue("FLOW_EMAIL");
  const password = requireValue("FLOW_PASSWORD");
  const name = requireValue("FLOW_NAME");
  const organizationName = requireValue("FLOW_ORGANIZATION");
  const projectName = requireValue("FLOW_PROJECT");
  const connection = new ConnectionConfig({ web: webEndpoint }, true);

  const registration = await RegisterUser(connection, email, password, name);
  requireSuccess(registration, "signup");

  const registrationData = registration.getData();
  const user = registrationData?.getUser();
  const token = registrationData?.getToken();
  if (!user || user.getEmail() !== email || !token?.getToken()) {
    throw new Error("signup returned incomplete authentication data");
  }

  const auth = {
    authorization: token.getToken(),
    "x-auth-id": user.getId(),
  };
  const organization = await CreateOrganization(
    connection,
    organizationName,
    "agency",
    "software",
    auth,
  );
  requireSuccess(organization, "organization creation");
  if (organization.getData()?.getName() !== organizationName) {
    throw new Error("organization creation returned unexpected data");
  }

  const project = await CreateProject(
    connection,
    projectName,
    "Created by the CI onboarding flow",
    auth,
  );
  requireSuccess(project, "project creation");
  if (project.getData()?.getName() !== projectName) {
    throw new Error("project creation returned unexpected data");
  }

  const authenticated = await AuthenticateUser(connection, email, password);
  requireSuccess(authenticated, "post-onboarding signin");
  const authenticatedData = authenticated.getData();
  const organizationID = organization.getData().getId();
  const projectID = project.getData().getId();
  if (authenticatedData?.getOrganizationrole()?.getOrganizationid() !== organizationID) {
    throw new Error("signin did not return the created organization role");
  }
  if (!authenticatedData.getProjectrolesList().some((role) => role.getProjectid() === projectID)) {
    throw new Error("signin did not return the created project role");
  }

  console.log("Signup, organization, and project flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);
