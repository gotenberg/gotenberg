Generate a valid certificate and private key:

```bash
# OpenSSL 1.1.1+
openssl req -x509 -newkey rsa:4096 -sha256 -days 9999 -nodes \
  -keyout key.pem -out cert.pem -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1,IP:::1"
```

Check a certificate:

```bash
openssl x509 -in cert.pem -text
```

Bundle them into a PKCS #12 store, `cert.p12`, with the password `secret`:

```bash
openssl pkcs12 -export -inkey key.pem -in cert.pem -out cert.p12 -passout pass:secret
```
