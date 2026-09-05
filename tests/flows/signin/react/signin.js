require("../../react-environment");

const {
  AuthenticateUser,
  ConnectionConfig,
} = require("@rapidaai/react");

const webEndpoint = process.env.WEB_API_REACT_ENDPOINT || "http://web-api:9001";
const email = process.env.FLOW_EMAIL;
const password = process.env.FLOW_PASSWORD;

function requireValue(value, name) {
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

function authenticate(connection, userEmail, userPassword) {
  return new Promise((resolve, reject) => {
    AuthenticateUser(connection, userEmail, userPassword, (error, response) => {
      if (error) {
        reject(error);
        return;
      }
      resolve(response);
    });
  });
}

async function main() {
  const connection = new ConnectionConfig({ web: webEndpoint }, true);
  const response = await authenticate(
    connection,
    requireValue(email, "FLOW_EMAIL"),
    requireValue(password, "FLOW_PASSWORD"),
  );

  if (!response?.getSuccess() || response.getCode() !== 200) {
    throw new Error(`signin failed with code ${response?.getCode()}`);
  }

  const user = response.getData()?.getUser();
  const token = response.getData()?.getToken();
  if (!user || user.getEmail() !== email) {
    throw new Error("signin returned an unexpected user");
  }
  if (!token?.getToken()) {
    throw new Error("signin returned no authentication token");
  }

  console.log("React SDK signin flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);
