FROM golang:1.24-alpine
ARG GID=100

RUN apk update --no-cache && \
  apk add rsync openssh sudo --no-cache

RUN addgroup -S test --gid ${GID} && \
    adduser --uid 1000 -S -s /bin/sh test -G test -h /home/test && \
    sed -i "s/^test:!:/test:\*/" /etc/shadow && \
    echo "test ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers && \
    mkdir -p /home/test/.ssh

COPY id_ed2551* /home/test/.ssh/
COPY end-to-end /home/test/end-to-end
COPY bundle /home/test/bundle
RUN touch /home/test/.ssh/known_hosts && chmod 600 /home/test/.ssh/known_hosts
RUN chown -R test:test /home/test && chmod 600 /home/test/.ssh/id_ed25519 && chmod 644 /home/test/.ssh/id_ed25519.pub && chmod 700 /home/test/.ssh

WORKDIR /home/test
USER test