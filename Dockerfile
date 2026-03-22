FROM golang:1.25-alpine@sha256:8e02eb337d9e0ea459e041f1ee5eece41cbb61f1d83e7d883a3e2fb4862063fa AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /protoc-gen-proto2pydantic .

FROM scratch
COPY --from=build /protoc-gen-proto2pydantic /protoc-gen-proto2pydantic
ENTRYPOINT ["/protoc-gen-proto2pydantic"]
