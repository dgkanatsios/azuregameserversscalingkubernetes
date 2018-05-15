
#build stage
FROM golang:1.10.1-alpine3.7 AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY . .
RUN cd cmd/eventhandler && go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app cmd/eventhandler

#final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/app .
CMD ["./app"]
LABEL Name=docker_openarena_k8s_eventhandler Version=0.0.2
