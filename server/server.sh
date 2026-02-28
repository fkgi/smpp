#! /bin/bash

echo -e 'HTTP/1.1 200 OK\r
Content-Type: application/json\r
Connection: close\r
\r
{
  "status": 0,
  "id": "dummy-id"
}' | nc -l 8082

# ./roundrobin -bind trx -http :8081 -tls localhost:2775
# ./roundrobin -tls -crt ~/server.crt -key ~/server.key