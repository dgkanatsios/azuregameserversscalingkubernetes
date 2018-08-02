
#build stage
FROM golang:1.10.1-alpine3.7 AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY . .
RUN cd cmd/apiserver && go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o apiserver ./cmd/apiserver

#final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/apiserver /build/html ./
CMD ["./apiserver"]
EXPOSE 8000
LABEL Name=docker_openarena_k8s_apiserver
