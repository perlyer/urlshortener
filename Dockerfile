# Многоступенчатая сборка Go-сервиса. Какой именно бинарь собрать - задаётся
# через build-arg SERVICE (api / redirector / analytics).

# ── сборка ──
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG SERVICE
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service ./cmd/${SERVICE}

# ── рантайм ──
FROM alpine:3.20
RUN adduser -D -u 10001 app
COPY --from=build /bin/service /bin/service
USER app
ENTRYPOINT ["/bin/service"]
