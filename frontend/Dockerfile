# syntax=docker/dockerfile:1

# Multi-platform (linux/amd64 and linux/arm64, see .github/workflows/docker.yml): the build output
# is static files, so the build stage runs on the builder's own platform; only the nginx stage is
# per platform.

## Build
FROM --platform=$BUILDPLATFORM node:24-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

## Deploy
FROM nginx:1.30-alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY --from=builder /app/set-env.sh /usr/share/nginx/html/set-env.sh
RUN chmod +x /usr/share/nginx/html/set-env.sh
COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80

CMD ["sh", "-c", "cd /usr/share/nginx/html && ./set-env.sh && nginx -g 'daemon off;'"]
