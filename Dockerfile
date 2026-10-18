FROM golang:1.27.2-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /task-manager ./cmd/api

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app \
    && apk --no-cache add ca-certificates

COPY --from=build /task-manager /usr/local/bin/task-manager

USER app
EXPOSE 9090

ENTRYPOINT ["/usr/local/bin/task-manager"]
