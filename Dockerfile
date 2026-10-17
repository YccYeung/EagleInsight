# Build stage: compiles a static binary, nothing from this stage ships in the final image.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY backend-go/go.mod backend-go/go.sum ./
RUN go mod download
COPY backend-go/ ./
# CGO_ENABLED=0 keeps the binary fully static (modernc.org/sqlite needs no cgo).
RUN CGO_ENABLED=0 go build -o /eagleinsight .

# Runtime stage: alpine rather than scratch, so there's a shell to exec into
# when debugging a running pod and so a non-root user can own its data dir.
FROM alpine:3.21
RUN apk add --no-cache ca-certificates && \
    addgroup -S app && adduser -S app -G app && \
    mkdir -p /data && chown app:app /data

COPY --from=build /eagleinsight /eagleinsight
COPY frontend /frontend

USER app
ENV FRONTEND_DIR=/frontend
ENV DB_PATH=/data/eagleinsight.db
# Hosts such as Cloud Run / Kubernetes tell the container which port to listen on through PORT.
ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/eagleinsight"]
