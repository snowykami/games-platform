FROM node:24-alpine AS web-build
WORKDIR /app

RUN corepack enable

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY web/package.json web/package.json
RUN pnpm install --frozen-lockfile

COPY web web
COPY server/internal/web server/internal/web
RUN pnpm --dir web build:embed

FROM golang:1.25-alpine AS server-build
WORKDIR /app

COPY go.work ./
COPY server/go.mod server/go.sum server/
RUN go work sync && go mod download -C server

COPY server server
COPY --from=web-build /app/server/internal/web/dist server/internal/web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -C server -trimpath -ldflags="-s -w" -o /out/games-platform ./cmd/api

FROM alpine:3.22
RUN addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app

ENV PORT=8901
EXPOSE 8901

COPY --from=server-build /out/games-platform /app/games-platform
ENTRYPOINT ["/app/games-platform"]
