
#build stage
FROM golang:1.10.1-stretch AS builder
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
LABEL Name=docker_openarena_k8s_eventhandler Version=0.0.1
