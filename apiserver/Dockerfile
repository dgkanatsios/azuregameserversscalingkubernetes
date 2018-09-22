#build stage
FROM golang:1.10.4-alpine3.8 AS builder
RUN apk add --no-cache git
WORKDIR /build
COPY . .
RUN cd cmd/apiserver && go get -d -v 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o apiserver ./cmd/apiserver

#final stage
FROM alpine:3.8
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/apiserver ./
COPY --from=builder /build/cmd/apiserver/html ./html
CMD ["./apiserver"]
EXPOSE 8000
LABEL Name=aks_gaming_apiserver
