FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o app ./cmd/main.go

RUN chmod 755 /app/app && chmod 644 /app/.env 

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=permissions /app/.env .env
COPY --from=permissions /app/app ./app

USER nonroot:nonroot

EXPOSE 8530

CMD ["./app"]