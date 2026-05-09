package server

import (
	"io"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/ashwaniYDV/go-redis/config"
	"github.com/ashwaniYDV/go-redis/core"
)

// toArrayString converts a slice of empty interfaces (as produced by the
// RESP array decoder) into a slice of strings. Each element is asserted
// as a string because RESP commands are arrays of bulk strings.
func toArrayString(ai []interface{}) ([]string, error) {
	as := make([]string, len(ai))
	for i := range ai {
		as[i] = ai[i].(string)
	}
	return as, nil
}

// readCommands reads from the connection and decodes one or more
// pipelined RESP commands. The buffer can contain multiple commands
// when a client uses pipelining.
func readCommands(c io.ReadWriter) (core.RedisCmds, error) {
	// TODO: Max read in one shot is 512 bytes
	// To allow input > 512 bytes, then repeated read until
	// we get EOF or designated delimiter
	var buf []byte = make([]byte, 512)
	n, err := c.Read(buf[:])
	if err != nil {
		return nil, err
	}

	values, err := core.Decode(buf[:n])
	if err != nil {
		return nil, err
	}

	var cmds core.RedisCmds = make(core.RedisCmds, 0)
	for _, value := range values {
		tokens, err := toArrayString(value.([]interface{}))
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, &core.RedisCmd{
			Cmd:  strings.ToUpper(tokens[0]),
			Args: tokens[1:],
		})
	}
	return cmds, nil
}

// respond evaluates the whole pipeline and writes every reply back
// to the client in a single Write (delegated to core.EvalAndRespond).
func respond(cmds core.RedisCmds, c io.ReadWriter) {
	core.EvalAndRespond(cmds, c)
}

func RunSyncTCPServer() {
	log.Println("starting a synchronous TCP server on", config.Host, config.Port)

	var con_clients int = 0

	// listening to the configured host:port
	lsnr, err := net.Listen("tcp", config.Host+":"+strconv.Itoa(config.Port))
	if err != nil {
		panic(err)
	}

	for {
		// blocking call: waiting for the new client to connect
		c, err := lsnr.Accept()
		if err != nil {
			panic(err)
		}

		// increment the number of concurrent clients
		con_clients += 1
		log.Println("client connected with address:", c.RemoteAddr(), "concurrent clients", con_clients)

		for {
			// over the socket, continuously read the commands and respond
			cmds, err := readCommands(c)
			if err != nil {
				c.Close()
				con_clients -= 1
				log.Println("client disconnected", c.RemoteAddr(), "concurrent clients", con_clients)
				if err == io.EOF {
					break
				}
				log.Println("err", err)
				break
			}

			respond(cmds, c)
		}
	}
}
