# AirMap Client

The goal of this exercise is to connect to a streaming grpc endpoint, consume the stream, and periodically emit rolling averages of metrics from the stream. 

By default, the AirMap client consumes the entire stream of data.  Optionally, you may pass in a bounding box via the command line with coordinates in the following order (sw lat, sw lon, ne lat, ne lon).

To run the client and consume the full stream:

    go run client.go

with a bounding box for greater LA:

    go run client.go -b "33.674069,-118.619385,34.420505,-117.993164"

for convenience, 3 locations are predefined (la, usa, eu)

    go run client.go -b la
