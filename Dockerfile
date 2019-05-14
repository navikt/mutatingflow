FROM alpine:3.9
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY mutatingflow /app/.
EXPOSE 8080
EXPOSE 8443
CMD ["/app/mutatingflow"]
