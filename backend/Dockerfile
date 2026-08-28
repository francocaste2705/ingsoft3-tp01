# ---- Etapa 1: build (con el SDK de Go) ----
FROM golang:1.22-alpine AS build
WORKDIR /src

COPY . .

# go mod tidy resuelve e instala las dependencias necesarias para compilar
# y genera el go.sum (no lo traiamos commiteado desde el vamos).
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# ---- Etapa 2: runtime minimo (sin el SDK) ----
FROM alpine:3.20 AS final
WORKDIR /app
COPY --from=build /app/server .
EXPOSE 8080
ENTRYPOINT ["./server"]