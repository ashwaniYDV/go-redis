//go:build darwin

package server

import (
	"log"
	"net"
	"syscall"
	"time"

	"github.com/ashwaniYDV/go-redis/config"
	"github.com/ashwaniYDV/go-redis/core"
)

var con_clients int = 0
var cronFrequency time.Duration = 1 * time.Second
var lastCronExecTime time.Time = time.Now()

func RunAsyncTCPServer() error {
	log.Println("starting an asynchronous TCP server on", config.Host, config.Port)

	max_clients := 20000

	// Create a socket
	serverFD, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(serverFD)

	// Set the Socket operate in a non-blocking mode
	if err = syscall.SetNonblock(serverFD, true); err != nil {
		return err
	}

	// Allow port reuse
	if err = syscall.SetsockoptInt(serverFD, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return err
	}

	// Bind the IP and the port
	ip4 := net.ParseIP(config.Host).To4()
	if err = syscall.Bind(serverFD, &syscall.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	}); err != nil {
		return err
	}

	// Start listening
	if err = syscall.Listen(serverFD, max_clients); err != nil {
		return err
	}

	// AsyncIO starts here!!

	// creating kqueue instance (macOS equivalent of epoll)
	kqFD, err := syscall.Kqueue()
	if err != nil {
		log.Fatal(err)
	}
	defer syscall.Close(kqFD)

	// Register the server FD for read events in kqueue
	changeEvent := syscall.Kevent_t{
		Ident:  uint64(serverFD),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
		Fflags: 0,
		Data:   0,
		Udata:  nil,
	}

	changeList := []syscall.Kevent_t{changeEvent}
	if _, err = syscall.Kevent(kqFD, changeList, nil, nil); err != nil {
		return err
	}

	// Buffer to hold events returned by kqueue
	events := make([]syscall.Kevent_t, max_clients)

	// timeout so the loop wakes up at least every cronFrequency to run the cron
	timeout := syscall.NsecToTimespec(cronFrequency.Nanoseconds())

	for {
		// run the active expiry cron in the event loop itself
		// (Redis is single threaded; we don't spawn a separate thread)
		if time.Now().After(lastCronExecTime.Add(cronFrequency)) {
			core.DeleteExpiredKeys()
			lastCronExecTime = time.Now()
		}

		// see if any FD is ready for an IO
		nevents, err := syscall.Kevent(kqFD, nil, events, &timeout)
		if err != nil {
			continue
		}

		for i := 0; i < nevents; i++ {
			// if the socket server itself is ready for an IO
			if int(events[i].Ident) == serverFD {
				// accept the incoming connection from a client
				fd, _, err := syscall.Accept(serverFD)
				if err != nil {
					log.Println("err", err)
					continue
				}

				// increase the number of concurrent clients count
				con_clients++
				syscall.SetNonblock(fd, true)

				// add this new TCP connection to be monitored
				socketClientEvent := syscall.Kevent_t{
					Ident:  uint64(fd),
					Filter: syscall.EVFILT_READ,
					Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
					Fflags: 0,
					Data:   0,
					Udata:  nil,
				}
				changeList := []syscall.Kevent_t{socketClientEvent}
				if _, err := syscall.Kevent(kqFD, changeList, nil, nil); err != nil {
					log.Fatal(err)
				}
			} else {
				comm := core.FDComm{Fd: int(events[i].Ident)}
				cmds, err := readCommands(comm)
				if err != nil {
					syscall.Close(int(events[i].Ident))
					con_clients -= 1
					continue
				}
				respond(cmds, comm)
			}
		}
	}
}
