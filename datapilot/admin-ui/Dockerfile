# Stage 1: Builder
FROM node:20-alpine AS builder

WORKDIR /app

COPY admin-ui/package.json ./
RUN npm install

COPY admin-ui/ .
RUN npm run build

# Stage 2: Runtime
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY admin-ui/nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
