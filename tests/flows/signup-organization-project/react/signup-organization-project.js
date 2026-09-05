require("../../react-environment");

const {
  AuthenticateUser,
  ConnectionConfig,
  CreateOrganization,
  CreateProject,
  RegisterUser,
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
    throw new Error(`${action} failed with code ${response?.getCode()}: ${message}`);
  }
}

async function main() {
  const email = requireValue("FLOW_EMAIL");
  const password = requireValue("FLOW_PASSWORD");
  const name = requireValue("FLOW_NAME");
  const organizationName = requireValue("FLOW_ORGANIZATION");
  const projectName = requireValue("FLOW_PROJECT");
  const connection = new ConnectionConfig({ web: webEndpoint }, true);

  const registration = await call((callback) =>
    RegisterUser(connection, email, password, name, callback),
  );
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
  const organization = await call((callback) =>
    CreateOrganization(connection, organizationName, "agency", "software", auth, callback),
  );
  requireSuccess(organization, "organization creation");
  if (organization.getData()?.getName() !== organizationName) {
    throw new Error("organization creation returned unexpected data");
  }

  const project = await call((callback) =>
    CreateProject(
      connection,
      projectName,
      "Created by the CI React SDK onboarding flow",
      auth,
      callback,
    ),
  );
  requireSuccess(project, "project creation");
  if (project.getData()?.getName() !== projectName) {
    throw new Error("project creation returned unexpected data");
  }

  const authenticated = await call((callback) =>
    AuthenticateUser(connection, email, password, callback),
  );
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

  console.log("React SDK signup, organization, and project flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);
