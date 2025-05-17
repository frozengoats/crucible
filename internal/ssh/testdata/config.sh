#!/bin/sh

addgroup -S test
adduser -S test -G test

mkdir -p /home/test/.ssh
chmod 700 /home/test/.ssh

cat /tmp/id_ed25519.pub >> /home/test/.ssh/authorized_keys

chmod 644 /home/test/.ssh/authorized_keys

chown -R test:test /home/test/.ssh

touch /home/test/done.file

echo "all done"