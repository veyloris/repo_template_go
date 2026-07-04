# Pin both stages by digest so Renovate can track and reproducible builds stay reproducible.
FROM golang:1.26-alpine@sha256:f85330846cde1e57ca9ec309382da3b8e6ae3ab943d2739500e08c86393a21b1 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG COMMIT=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/myorg/myapp/internal/version.Commit=${COMMIT}" \
    -o myapp ./cmd/myapp/

FROM gcr.io/distroless/static-debian12:nonroot@sha256:d093aa3e30dbadd3efe1310db061a14da60299baff8450a17fe0ccc514a16639

COPY --from=builder /build/myapp /myapp

USER nonroot:nonroot

ENTRYPOINT ["/myapp"]
CMD ["serve"]
