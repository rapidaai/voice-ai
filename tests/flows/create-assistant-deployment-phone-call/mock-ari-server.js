const http = require("node:http");

function startMockARIServer(port) {
  return new Promise((resolve, reject) => {
    const server = http.createServer((request, response) => {
      if (request.method === "GET" && request.url === "/health") {
        response.writeHead(200, { "Content-Type": "application/json" });
        response.end(JSON.stringify({ status: "ok" }));
        return;
      }

      if (
        request.method === "POST" &&
        request.url?.startsWith("/ari/channels?")
      ) {
        let body = "";
        request.setEncoding("utf8");
        request.on("data", (chunk) => {
          body += chunk;
        });
        request.on("end", () => {
          const url = new URL(request.url, "http://localhost");
          const expectedAuthorization = `Basic ${Buffer.from(
            "ci-flow:ci-flow",
          ).toString("base64")}`;
          const variables = body ? JSON.parse(body).variables : {};
          const isValid =
            request.headers.authorization === expectedAuthorization &&
            url.searchParams.get("endpoint") ===
              `PJSIP/${process.env.FLOW_TO_NUMBER}` &&
            url.searchParams.get("callerId") === process.env.FLOW_FROM_NUMBER &&
            url.searchParams.get("app") === "rapida" &&
            url.searchParams
              .get("appArgs")
              ?.startsWith("incoming,assistant_id=") &&
            variables?.RAPIDA_CONTEXT_ID;

          if (!isValid) {
            response.writeHead(400, { "Content-Type": "application/json" });
            response.end(JSON.stringify({ message: "unexpected ARI request" }));
            return;
          }

          response.writeHead(201, { "Content-Type": "application/json" });
          response.end(JSON.stringify({ id: `ci-asterisk-${Date.now()}` }));
        });
        return;
      }

      response.writeHead(404, { "Content-Type": "application/json" });
      response.end(JSON.stringify({ message: "not found" }));
    });

    server.once("error", reject);
    server.listen(port, "0.0.0.0", () => resolve(server));
  });
}

if (require.main === module) {
  const port = Number(process.argv[2]);
  if (!Number.isInteger(port) || port <= 0) {
    console.error("mock ARI server requires a valid port");
    process.exit(1);
  }
  startMockARIServer(port).catch((error) => {
    console.error(error);
    process.exit(1);
  });
}

module.exports = { startMockARIServer };
