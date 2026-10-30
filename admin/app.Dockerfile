# Production-style image for the Admin app. Build context is admin/. The stack
# passes the build arguments; the defaults are the in-network names the stack uses.
# Vite bakes VITE_* in at build time, so changing one needs a rebuild.

FROM oven/bun:1.4.2 AS build-stage

WORKDIR /app

COPY package.json bun.lock .npmrc ./
RUN bun install --frozen-lockfile

COPY . .

# Empty values are tolerated by the build: it warns, and the app runs without them.
ARG VITE_API_BASE_URL=http://api:8080
ARG VITE_CLERK_PUBLISHABLE_KEY=
ARG VITE_SITE_URL=
ARG VITE_IMAGE_HOST=
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL \
    VITE_CLERK_PUBLISHABLE_KEY=$VITE_CLERK_PUBLISHABLE_KEY \
    VITE_SITE_URL=$VITE_SITE_URL \
    VITE_IMAGE_HOST=$VITE_IMAGE_HOST

RUN bun run build

FROM oven/bun:1.4.2-slim AS build-release-stage

WORKDIR /app

# The serve script reads vercel.json from its own package root, so the layout matches the source tree.
COPY --from=build-stage /app/dist/client ./dist/client
COPY --from=build-stage /app/tools/serve.ts ./tools/serve.ts
COPY --from=build-stage /app/vercel.json ./vercel.json

ENV PORT=3001
EXPOSE 3001

USER bun

CMD ["bun", "tools/serve.ts"]
