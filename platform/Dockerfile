FROM node:22-alpine AS frontend

WORKDIR /app/web
COPY web/package.json ./
RUN npm install --include=dev
COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /usr/local/bin/platform ./cmd/server

FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget
COPY --from=backend /usr/local/bin/platform /usr/local/bin/platform
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/platform"]