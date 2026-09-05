const {
  AuthenticateUser,
  ConnectionConfig,
} = require("@rapidaai/nodejs");

const webEndpoint = process.env.WEB_API_ENDPOINT || "web-api:9001";
const email = process.env.FLOW_EMAIL;
const password = process.env.FLOW_PASSWORD;

function requireValue(value, name) {
  if (!value) {
    throw new Error(`${name} is required`);
  }
  return value;
}

async function main() {
  const connection = new ConnectionConfig({ web: webEndpoint }, true);
  const response = await AuthenticateUser(
    connection,
    requireValue(email, "FLOW_EMAIL"),
    requireValue(password, "FLOW_PASSWORD"),
  );

  if (!response.getSuccess() || response.getCode() !== 200) {
    throw new Error(`signin failed with code ${response.getCode()}`);
  }

  const authentication = response.getData();
  const user = authentication?.getUser();
  const token = authentication?.getToken();
  if (!user || user.getEmail() !== email) {
    throw new Error("signin returned an unexpected user");
  }
  if (!token?.getToken()) {
    throw new Error("signin returned no authentication token");
  }

  console.log("Signin flow passed");
}

main().then(
  () => process.exit(0),
  (error) => {
    console.error(error);
    process.exit(1);
  },
);
