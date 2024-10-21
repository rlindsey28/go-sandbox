#Build stage
#You can simple add the CMD ["./pub-service-app"] to the end of this stage
#and skip the deploy stage to run the app but it will be a much larger image
FROM golang:1.23-alpine AS builder

WORKDIR /

COPY . .
RUN go mod download

RUN go build -o pub-service-app .

#Deploy stage
#Doing a multi-stage build will keep the image size small
FROM alpine:latest

WORKDIR /app

COPY --from=builder /pub-service-app .

CMD ["./pub-service-app"]
