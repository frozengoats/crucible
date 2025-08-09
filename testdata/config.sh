#!/bin/sh
set -e

addgroup -S test
adduser --uid 1000 -S -s /bin/sh test -G test -h /home/test
sed -i "s/^test:!:/test:\*/" /etc/shadow
echo "test ALL=(ALL) NOPASSWD: ALL" >> /etc/sudoers
mkdir -p /home/test/.ssh
chmod 700 /home/test/.ssh

cat /tmp/id_ed25519.pub >> /home/test/.ssh/authorized_keys
cat /tmp/id_ed25519_passphrase.pub >> /home/test/.ssh/authorized_keys

chmod 644 /home/test/.ssh/authorized_keys
chown -R test:test /home/test

addgroup -S phonk
adduser --uid 1001 -S -s /bin/sh phonk -G phonk -h /home/phonk
chown -R phonk:phonk /home/phonk

touch /home/test/done.file
echo "all done"