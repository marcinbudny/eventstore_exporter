openssl genrsa -out ca/rootCA.key 4096

openssl req \
    -x509 \
    -new \
    -nodes \
    -key ca/rootCA.key \
    -sha256 \
    -days 10000 \
    -subj "/O=eventstore_exporter" \
    -out ca/rootCA.crt

openssl genrsa -out node/eventstoredb-node.key 2048

openssl req \
    -new \
    -sha256 \
    -key node/eventstoredb-node.key \
    -subj "/CN=eventstoredb-node" \
    -out node/eventstoredb-node.csr

openssl x509 \
    -req \
    -sha256 \
    -days 10000 \
    -in node/eventstoredb-node.csr \
    -CA ca/rootCA.crt -CAkey ca/rootCA.key \
    -CAcreateserial \
    -extfile <(\
    printf "
        authorityKeyIdentifier=keyid,issuer
        basicConstraints=CA:FALSE
        keyUsage = digitalSignature, nonRepudiation, dataEncipherment
        subjectAltName = IP:172.16.1.11,IP:172.16.1.12,IP:172.16.1.13,IP:172.16.1.14,IP:172.16.1.15"
    ) \
    -out node/eventstoredb-node.crt

rm node/eventstoredb-node.csr
