# syntax=docker/dockerfile:1
FROM golang:1.27.0-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/timesheet ./cmd/timesheet

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/timesheet /usr/local/bin/timesheet
WORKDIR /app
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/timesheet"]
CMD ["--help"]
