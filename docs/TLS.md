# TLS

## Certbot

Obtain or renew a certificate, but do not install it:

```console
$ certbot certonly --standalone -d example.org
Saving debug log to /var/log/letsencrypt/letsencrypt.log
Plugins selected: Authenticator standalone, Installer None
Obtaining a new certificate
Performing the following challenges:
http-01 challenge for example.org
Cleaning up challenges
[...]
```
