# GRPC Health Checker 

This repository provides all the keys to deploy a prometheus exporter that provides metrics on GRPC server health.

## Running

To run the prometheus exporter, you can use the following binary `healthchecker`. 
Provide the GRPC server endpoints you want to health check as arguments of this command. 

To check all the available options, you can run `healthchecker --help`.

```bash 
 healthChecker mainnet.eth.streamingfast.io,mainnet.btc.streamingfast.io -p /sf.blockmeta.v2.Block/Head -H "Authorization: Bearer $TOKEN" 
```




