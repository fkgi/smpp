# Round-Robin SMPP debugger
Round-Robin can act as ESME or SMSC for SMPP.

Round-Robin can send SMPP request to peer node.
It make SMPP request mesage from received HTTP REST request, then send the message to peer node.
It receive SMPP answer message, then make HTTP REST answer from received SMPP answer message.

Round-Robin can receive SMPP request from peer node.
It make HTTP REST request from received SMPP request message, then send the request to pre-configured HTTP server.
It receive HTTP REST answer, then make SMPP answer message from received HTTP REST answer.

HTTP REST request/answer must have specific format JSON document.

# How to run Round-Robin
Commandline options.

```
roundrobin [OPTION]... [[IP]:PORT]
```

Commandline example

```
roundrobin -s test -i :8080 -b mockserver:8080 -d trx -p password -y test 192.168.10.10:2775
```

## Args
- `IP`  
SMPP peer address when act as ESME, or SMPP local address when act as SMSC.

- `PORT`  
SMPP peer port when act as ESME, or SMPP local port when act as SMSC.(default :2775)

## Options
- `-s`  
System ID of SMPP bind. Value is any string but must not empty.

- `-i`  
Local listening address and port for receiving HTTP REST request.
Value must have format `host[:port]`.
`host` is hostname or IP address.
IP address is resolved from hostname if hostname is specified.
`port` is port number.

- `-b`  
Peer address and port for sending HTTP REST request.
Value must have format `host[:port]`.
`host` is hostname or IP address.
IP address is resolved from hostname if hostname is specified.
`port` is port number.

- `-d`  
Bind type.
Value must have `tx` or `rx` or `trx` for client. Or value must have `svr` for server.
Default value is `svr`.

- `-p`  
Password for ESME authentication.

- `-y`  
Type of ESME system. Value is any string. Default value is `ROUNDROBIN`.

- `-o`  
Type of Number for ESME address. Digits value.

- `-n`  
Numbering Plan Indicator for ESME address. Digits value.

- `-a`  
UNIX Regular Expression notation of ESME address.

- `-t`  
If this parameter is enabled, TLS is enabled.

- `-c`  
TLS crt file when act as server and TLS is enabled.

- `-k`  
TLS key file when act as server and TLS is enabled.

- `-h`  
Print usage.

# Format of REST message
Only POST method is acceptable for HTTP REST request.
HTTP URI path has prefix `/smppmsg/v1`.
HTTP URI path has SMPP message name by `/data` or `/deliver` or `/submit`. 
```
POST http://roundrobin:8080/smppmsg/v1/submit
```

HTTP body is JSON Map object.

SMPP deliver request.
```
{
    "svc_type": "svc1",
    "src_ton": 1,
    "src_npi": 1,
    "src_addr": "123",
    "dst_ton": 1,
    "dst_npi": 1,
    "dst_addr": "987",
    "esm_class": 1,
    "protocol_id": 1,
    "priority_flag": 1,
    "schedule_delivery_time": "",
    "validity_period": "",
    "registered_delivery": 1,
    "replace_if_present_flag": 1,
    "data_coding": 1,
    "sm_default_sm_id": 1,
    "short_message": "",
    "options": {}
}
```

SMPP deliver response.
```
{
    "status": "ESME_ROK",
    "id": "generated ID"
}
```

SMPP data request.
```
{
    "svc_type": "svc1",
    "src_ton": 0,
    "src_npi": 0,
    "src_addr": "123",
    "dst_ton": 0,
    "dst_npi": 0,
    "dst_addr": "987",
    "esm_class": 0,
    "registered_delivery": 0,
    "data_coding": 0,
    "options": {
        "1060": "dGVzdGRhdGE="
    }
}
```

SMPP data response.
```
{
    "status": "ESME_ROK",
    "id": "generated ID",
    "options": {}
}
```