import { defineRailway, github, preserve, project, service } from "railway/iac";

export default defineRailway(() => {
  const api = service("api", {
    source: github("itsLeonB/cardstack", { checkSuites: true, rootDirectory: "backend" }),
    build: { buildEnvironment: "V3", builder: "DOCKERFILE", dockerfilePath: "Dockerfile", watchPatterns: ["/backend/**"] },
    healthcheck: "/health",
    preDeploy: "/migrate",
    replicas: { "asia-southeast1-eqsg3a": 1 },
    deploy: { limitOverride: { containers: { cpu: 0.5, memoryBytes: 500000000 } }, restartPolicyMaxRetries: 3, sleepApplication: true },
    networking: { privateNetworkEndpoint: "cardstack" },
    env: { APP_CLIENT_URLS: preserve(), APP_ENV: preserve(), APP_PORT: preserve(), APP_TIMEOUT: preserve(), AUTH_COOKIE_DOMAIN: preserve(), AUTH_COOKIE_SAMESITE: preserve(), AUTH_COOKIE_SECURE: preserve(), AUTH_JWT_DURATION: preserve(), AUTH_JWT_ISSUER: preserve(), AUTH_JWT_SECRET: preserve(), AUTH_REFRESH_TOKEN_TTL: preserve(), DB_CONN_MAX_LIFETIME: preserve(), DB_HOST: preserve(), DB_MAX_IDLE_CONNS: preserve(), DB_MAX_OPEN_CONNS: preserve(), DB_NAME: preserve(), DB_PASSWORD: preserve(), DB_PORT: preserve(), DB_USER: preserve(), OTEL_ENABLED: preserve(), OTEL_SERVICE_NAME: preserve() },
  });

  return project("cardstack", {
    resources: [api],
  });
});
