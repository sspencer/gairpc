# AirMap Client

The goal of this exercise is to connect to a streaming grpc endpoint, consume the stream, and periodically emit rolling averages of metrics from the stream. 

By default, the AirMap client consumes the entire stream of data.  Optionally, you may pass in a bounding box via the command line with coordinates in the following order (sw lat, sw lon, ne lat, ne lon).

To run the client and consume the full stream:

    go run client.go

with a bounding box for greater LA:

    go run client.go -b "33.674069,-118.619385,34.420505,-117.993164"

for convenience, 3 locations are predefined (la, usa, eu)

    go run client.go -b la

## Redis-Like Fun

For fun, integrated a REDIS like interface into the AirMap Client.  After starting the client, open a new terminal and connect with redis-cli.  

### Commands

* **FLIGHTS** - get list of all flights being tracked
* **GET** - get latest data on specified flight
* **STATS** - get average X velocity, Y vecolity, altitude and more

### Example Session

    $ redis-cli -p 6060
    127.0.0.1:6060> get N341SP-599
"{\"trackId\":\"N341SP-599\",\"xvel\":-80,\"yvel\":26,\"alt\":1828.8000000000002,\"latitude\":33.86285,\"longitude\":-118.27153}"
    127.0.0.1:6060> stats
"{\"flightCount\":33,\"statCount\":59,\"xvelAvg\":82.18333333333334,\"yvelAvg\":66.38333333333334,\"altAvg\":1132.1330000000003}"
    127.0.0.1:6060> flights
     1) "N665PD-1569520683-adhoc-0"
     2) "N341SP-599"
     3) "SWA1951-776"