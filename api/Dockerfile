
#build stage
FROM golang:1.10.1-alpine3.7 AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY . .
RUN go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

#final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/app .
CMD ["./app"]
EXPOSE 8000
LABEL Name=docker_openarena_k8s_api Version=0.0.1
