import { defineRailway, github, preserve, project, service } from "railway/iac";

export default defineRailway(() => {
  const api = service("api", {
    source: github("itsLeonB/cardstack", { checkSuites: true, rootDirectory: "backend" }),
    build: { buildEnvironment: "V3", builder: "DOCKERFILE", dockerfilePath: "Dockerfile", watchPatterns: ["/backend/**"] },
    healthcheck: "/health",
    replicas: { "asia-southeast1-eqsg3a": 1 },
    // preDeployCommand runs /migrate (built alongside /api in the same
    // Dockerfile stage — see cmd/job) before each deploy goes live. Without
    // it, nothing in the pipeline ever applies schema migrations: cmd/api
    // never runs them itself, and a failing preDeployCommand blocks the
    // deploy rather than shipping a build that 500s on every DB query.
    deploy: { limitOverride: { containers: { cpu: 0.5, memoryBytes: 500000000 } }, restartPolicyMaxRetries: 3, sleepApplication: true, preDeployCommand: ["/migrate"] },
    networking: { privateNetworkEndpoint: "cardstack" },
    env: { APP_CLIENT_URLS: preserve(), APP_ENV: preserve(), APP_PORT: preserve(), APP_TIMEOUT: preserve(), DB_CONN_MAX_LIFETIME: preserve(), DB_HOST: preserve(), DB_MAX_IDLE_CONNS: preserve(), DB_MAX_OPEN_CONNS: preserve(), DB_NAME: preserve(), DB_PASSWORD: preserve(), DB_PORT: preserve(), DB_USER: preserve(), OTEL_ENABLED: preserve(), OTEL_SERVICE_NAME: preserve(), AUTH_JWT_SECRET: preserve(), AUTH_JWT_ISSUER: preserve(), AUTH_JWT_DURATION: preserve(), AUTH_REFRESH_TOKEN_TTL: preserve(), AUTH_COOKIE_DOMAIN: preserve(), AUTH_COOKIE_SECURE: preserve(), AUTH_COOKIE_SAMESITE: preserve() },
  });

  return project("cardstack", {
    resources: [api],
  });
});
