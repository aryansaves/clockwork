package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

func main() {
	var wg sync.WaitGroup
	listen, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	conn, err := listen.Accept()
	if err != nil {
		log.Fatal(err)
	}
	wg.Add(1)
	go handleconn(conn, &wg)
	wg.Wait()
}

func handleconn(conn net.Conn, wg *sync.WaitGroup ){
	defer conn.Close()
	defer wg.Done()
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF{
				break
			}
			fmt.Println("err ", err)
		}
		if line[0] == '*'{
			var data string
			args, err := strconv.Atoi(strings.TrimSpace(line[1:]))
			if err != nil {
				fmt.Printf("atoi line error %s",err)
				break
			}
			for range args*2 {
				arguements, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF{
						break
					}
					fmt.Println("err ", err)
				}
				if arguements[0] == '$'{
					continue
				}
				data = strings.TrimSpace(arguements)
				fmt.Printf("%s\n",data)
			}
			resp := decideResp(data)
			conn.Write([]byte(resp))
		}
	}	
}

func decideResp(data string)(string){
	switch data{
		case "PING":
			return "+PONG\r\n"
		default:
			return "+65\r\n"
	}
}