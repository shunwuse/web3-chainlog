# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/backend .

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/backend /backend
EXPOSE 8080
ENTRYPOINT ["/backend"]
