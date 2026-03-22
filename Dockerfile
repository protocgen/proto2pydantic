FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /protoc-gen-proto2pydantic .

FROM scratch
COPY --from=build /protoc-gen-proto2pydantic /protoc-gen-proto2pydantic
ENTRYPOINT ["/protoc-gen-proto2pydantic"]
