/*
   Mosh SRV, a wrapper around mosh (and eventually SSH), using SRV records
   Copyright (C) 2021  Clément Joly

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

type ConnectionType int

const (
	Mosh ConnectionType = iota
	Ssh  ConnectionType = iota
)

func help(args []string) {
	fmt.Printf("Mosh SRV\n%s hostname [mosh arguments...]\n\nThe hostname argument is required.\n", args[0])
	os.Exit(255)
}

func querySRV(connection ConnectionType, hostname string) (srvs []*net.SRV, err error) {
	switch connection {
	case Mosh:
		_, addrs, err := net.LookupSRV("mosh", "udp", hostname)
		if err != nil {
			return srvs, err
		}
		return addrs, nil
	case Ssh:
		net.LookupSRV("ssh", "tcp", hostname)
		log.Fatalln("SSH connection is not implemented yet")
		return srvs, err
	default:
		log.Fatalln("Unknown connection type")
		return srvs, err
	}
}

// callShell calls the remote shell
func callShell(connection ConnectionType, srv *net.SRV, shellArgs []string) error {
	shell := ""
	switch connection {
	case Mosh:
		shell = "mosh"
		// shell = "ssh"
	case Ssh:
		shell = "ssh"
	default:
		log.Fatalln("Unknown connection type")
	}

	// Remove the last dot in the Target
	var b strings.Builder
	if last := len(srv.Target) - 1; srv.Target[last] == '.' {
		b.WriteString(srv.Target[:last])
	} else {
		b.WriteString(srv.Target)
	}
	hostname := b.String()

	args := []string{}
	args = append(args, shellArgs...)
	// -p is used by both SSH and Mosh to set port
	args = append(args, "-p", fmt.Sprintf("%v:%v", srv.Port, srv.Port+1000))
	// args = append(args, "-p", fmt.Sprintf("%v", srv.Port))
	args = append(args, "--no-ssh-pty")
	args = append(args, hostname)

	fmt.Println("args", args)

	cmd := exec.Command(shell, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("execute %#v\n", cmd)
	log.Printf("EXECUTE %v\n", cmd)
	return cmd.Run()
}

func main() {
	args := os.Args
	if len(args) < 2 {
		help(args)
	}

	hostname := args[1]
	connection := Mosh
	shellArgs := make([]string, len(args)-2)
	if len(args) > 2 {
		copy(shellArgs, args[2:])
	}

	fmt.Println("args", args)

	srvs, err := querySRV(connection, hostname)
	if err != nil {
		log.Fatalln("Error querying SRV records:", err)
	}

	fmt.Println("SRVs", srvs)

	for _, srv := range srvs {
		fmt.Println("Trying host", srv.Target)
		err := callShell(connection, srv, shellArgs)
		if err != nil {
			if err, ok := err.(*exec.ExitError); ok {
				log.Println("INSIDE", err)
				continue
			}
			log.Fatal(err)
		}
	}
}
