
#build stage
FROM golang:1.10.1-alpine3.7 AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY . .
RUN cd cmd/controller && go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o controller ./cmd/controller

#final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/controller .
CMD ["./controller"]
LABEL Name=aks_gaming_controller
