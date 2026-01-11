FROM alpine:3 AS builder

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /ca-certificates.crt
COPY bin/crucible /crucible
ENTRYPOINT [ "/crucible" ]