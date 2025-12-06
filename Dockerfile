# syntax=docker/dockerfile:1

FROM golang:tip-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o blog-server ./main.go

FROM gcr.io/distroless/base-debian12
WORKDIR /app
ENV GIN_MODE=release
COPY --from=build /src/blog-server /app/blog-server
EXPOSE 8080
CMD ["/app/blog-server"]
