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
			cmdArgs := []string {} 
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
				cmdArgs = append(cmdArgs, data)
				fmt.Printf("%s\n",data)
			}
			resp := decideResp(cmdArgs)
			conn.Write([]byte(resp))
		}
	}	
}

func decideResp(cmdArgs[] string)(string){
	data := cmdArgs[0]
	switch data{
		case "ping", "PING":
			return "+PONG\r\n"
		case "echo", "ECHO":
			return "+" + cmdArgs[1] + "\r\n"
		case "set", "SET":
			return setData(cmdArgs)
		case "get", "GET":
			return getData(cmdArgs)
		case "config", "CONFIG":
			return "*0\r\n"
		default:
			return "+65\r\n"
	}
}

var (
	data = make(map[string]string)
	dataMutex sync.RWMutex
)
func getData(cmdArgs[] string) (string){
	dataMutex.Lock()
	value, ok := data[cmdArgs[1]]
	dataMutex.Unlock()
	if ok {
		return "+" + value + "\r\n"
	} else {
		return "$-1\r\n"
	}
}

func setData(cmdArgs[] string) (string){
	dataMutex.Lock()
	data[cmdArgs[1]] = cmdArgs[2]
	dataMutex.Unlock()
	return "+OK\r\n"
}
