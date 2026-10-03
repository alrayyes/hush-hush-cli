# Multi-stage: compile in a full toolchain image, copy only the binary into
# the runtime stage. Distroless because the build is static - there's no
# libc to bring along, and nothing left in the image to exec into if it's
# ever reached from outside.
FROM golang:1.27.1-bookworm@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static, so the distroless base below is enough. -trimpath keeps build
# machine paths out of the binary.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/hush-hush-cli ./cmd/hush-hush-cli

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab

COPY --from=build /out/hush-hush-cli /hush-hush-cli

# The :nonroot base image variant already sets the user, so this is
# explicit rather than load-bearing.
USER nonroot:nonroot

ENTRYPOINT ["/hush-hush-cli"]
