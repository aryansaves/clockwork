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
	listen, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}
	for {
	conn, err := listen.Accept()
	if err != nil {
		log.Println("accept line ", err)
		continue
	}
	go handleconn(conn)
	}
}

func handleconn(conn net.Conn){
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF{
				break
			}
			fmt.Println("err ", err)
		}
		if line[0] == '*' && len(line) > 0 {
			args, err := strconv.Atoi(strings.TrimSpace(line[1:]))
			if err != nil {
				fmt.Printf("atoi line error %s",err)
				break
			}
			cmdArgs := []string {}
			innerLoopFailed := false
			for range args*2 {
				arguements, err := reader.ReadString('\n')
				if err != nil {
					innerLoopFailed = true
					break
				}
				if len(arguements) > 0 && arguements[0] == '$'{
					continue
				}
				cmdArgs = append(cmdArgs, strings.TrimSpace(arguements))
				fmt.Printf("%s\n",strings.TrimSpace(arguements))
			}
			if innerLoopFailed || len(cmdArgs) == 0 {
				break
			}
			resp := decideResp(cmdArgs)
			_, err = conn.Write([]byte(resp))
			if err != nil {
				break
			}
		}
	}
}

func decideResp(cmdArgs[] string)(string){
	if len(cmdArgs) == 0 {
			return "-ERR empty command\r\n"
		}
	switch strings.ToUpper(cmdArgs[0]){
		case "PING":
			return "+PONG\r\n"
			
		case "ECHO":
			if len(cmdArgs) < 2 {
				return "-ERR wrong number of arguements for 'echo' command\r\n"
			}
			return "+" + cmdArgs[1] + "\r\n"
			
		case "EXISTS":
			if len(cmdArgs) < 2 {
				return "-ERR wrong number of arguements for 'echo' command\r\n"
			}
			count := 0
			dataMutex.RLock()
			for _, key := range cmdArgs[1:]{
				if _, exists := data[key]; exists {
					count++
				}
			}
			dataMutex.RUnlock()
			return ":" + strconv.Itoa(count) + "\r\n"
		case "DEL" :
			if len(cmdArgs) < 2 {
				return "-ERR wrong number of arguments for 'set' command\r\n"
			}
			count := 0
			dataMutex.Lock()
			for _, key := range cmdArgs[1:]{
				if _, exists := data[key]; exists {
					count++
					delete(data, key)
				}	
			}
			dataMutex.Unlock()
			return ":" + strconv.Itoa(count) + "\r\n"
		case "SET":
			if len(cmdArgs) < 3 {
				return "-ERR wrong number of arguments for 'set' command\r\n"
			}
			return setData(cmdArgs)
			
		case "GET":
			if len(cmdArgs) < 2 {
				return "-ERR wrong number of arguments for 'get' command\r\n"
			}
			return getData(cmdArgs)
			
		case "CONFIG":
			return "*-1\r\n"
			
		default:
			return "-ERR unknown command\r\n"
	}
}

var (
	data = make(map[string]string)
	dataMutex sync.RWMutex
)
func getData(cmdArgs[] string) (string){
	dataMutex.RLock()
	value, ok := data[cmdArgs[1]]
	dataMutex.RUnlock()
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
