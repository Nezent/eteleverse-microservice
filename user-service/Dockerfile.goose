FROM golang:1.25-alpine AS goose
WORKDIR /goose
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

FROM golang:1.25-alpine
WORKDIR /app
COPY --from=goose /go/bin/goose /usr/local/bin/goose
COPY . .
# Entrypoint and CMD will be set in the migration service
