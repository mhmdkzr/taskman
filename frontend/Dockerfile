FROM denoland/deno:2.9.4@sha256:c777b4b225501a61074837e90a826a58f99124837824023cd60334b1e2374498 AS builder

ENV DENO_DIR=/deno-dir

COPY frontend/.env /app/.env
WORKDIR /app/frontend

COPY frontend/package.json frontend/deno.lock frontend/.npmrc ./
RUN deno install

COPY frontend/ .
RUN deno task build

FROM denoland/deno:alpine-2.9.4@sha256:13851184d6705150b8b230c5377f26c3bd182865d28700bb72bc0b2c271b504a

ENV DENO_DIR=/deno-dir
WORKDIR /app

COPY --from=builder /app/frontend/.deno-deploy ./.deno-deploy
COPY --from=builder /deno-dir /deno-dir

USER deno

EXPOSE 8000

CMD ["deno", "run", "-A", ".deno-deploy/server.ts"]
