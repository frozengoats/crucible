FROM golang:1.24-alpine
ARG GID=100

RUN apk update --no-cache && \
  apk add rsync openssh sudo --no-cache

RUN addgroup -S test --gid ${GID} && \
    adduser --uid 1000 -S -s /bin/sh test -G test -h /home/test && \
    sed -i "s/^test:!:/test:\*/" /etc/shadow && \
    echo "test ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers && \
    mkdir -p /home/test/.ssh

COPY id_ed25519 /home/test/.ssh/
COPY end-to-end /home/test/end-to-end
RUN chown -R test:test /home/test

WORKDIR /home/test
USER test