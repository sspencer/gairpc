package airmap

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/tidwall/redcon"
)

func (f *Flights) Redislike(addr string) {
	log.Printf("started server at %s", addr)
	if err := redcon.ListenAndServe(addr, redHandler(f), nil, nil); err != nil {
		log.Fatal(err)
	}
}

func redHandler(f *Flights) func(conn redcon.Conn, cmd redcon.Command) {
	return func(conn redcon.Conn, cmd redcon.Command) {
		switch strings.ToLower(string(cmd.Args[0])) {
		default:
			conn.WriteError("ERR unknown command '" + string(cmd.Args[0]) + "'")
		case "ping":
			conn.WriteString("PONG")
		case "quit":
			conn.WriteString("OK")
			conn.Close()

		case "stats":
			if len(cmd.Args) != 1 {
				conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
				return
			}

			f.RLock()
			fs := f.flightStats
			f.RUnlock()

			b, _ := json.Marshal(fs)
			conn.WriteBulk(b)

		case "get":
			if len(cmd.Args) != 2 {
				conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
				return
			}

			f.RLock()
			fd, ok := f.flightMap[string(cmd.Args[1])]
			f.RUnlock()
			if !ok {
				conn.WriteNull()
			} else {
				b, _ := json.Marshal(fd)
				conn.WriteBulk(b)
			}

		case "flights":
			if len(cmd.Args) != 1 {
				conn.WriteError("ERR wrong number of arguments for '" + string(cmd.Args[0]) + "' command")
				return
			}

			f.RLock()
			m := f.flightMap
			cnt := len(m)
			ids := make([]string, cnt)
			i := 0
			for key := range m {
				ids[i] = key
				i++
			}
			f.RUnlock()

			conn.WriteArray(cnt)
			for _, id := range ids {
				conn.WriteBulkString(id)
			}
		}
	}
}
