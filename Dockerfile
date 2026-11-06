# API mode only: no Chrome in the image. The browser source needs a desktop and stays local.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/threads-leads ./cmd/app

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/threads-leads /usr/local/bin/threads-leads
# Same uid as the host user that owns ./data.
USER 1000:1000
ENTRYPOINT ["threads-leads"]
CMD ["run", "-api", "-every", "60m"]
