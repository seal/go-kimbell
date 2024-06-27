FROM golang as builder

WORKDIR /app

COPY . .

RUN go get .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# deployment image
FROM alpine:latest  
RUN apk --no-cache add ca-certificates

LABEL author="Will Kimbell"

WORKDIR /root/
COPY --from=builder /app/app .
COPY --from=builder /app/json ./json
COPY --from=builder /app/posts ./posts
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
CMD ["./app"]

EXPOSE 7002


