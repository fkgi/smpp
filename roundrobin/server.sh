#! /bin/bash

echo -e 'HTTP/1.1 200 OK\r
Content-Type: application/json\r
Connection: close\r
\r
{
  "status": 0,
  "id": "dummy-id"
}' | nc -l 8082
