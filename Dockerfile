FROM golang:1.26.4-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY main.go ./
COPY dump.json ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o clockwork .

FROM scratch
COPY --from=builder /app/clockwork .
COPY --from=builder /app/dump.json .

EXPOSE 6379

ENTRYPOINT [ "./clockwork" ]