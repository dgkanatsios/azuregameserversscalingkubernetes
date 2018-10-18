#build stage
FROM golang:1.10.4-alpine3.8 AS builder
RUN apk add --no-cache git
WORKDIR /go/src/github.com/dgkanatsios/azuregameserversscalingkubernetes
COPY . .
RUN cd ./controller/cmd/controller 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/controller ./controller/cmd/controller

#final stage
FROM alpine:3.8
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/controller .
CMD ["./controller"]
