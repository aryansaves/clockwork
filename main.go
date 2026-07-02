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
	"time"
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
			
		case "INCR" :
		if len(cmdArgs) < 2 {
				return "-ERR wrong number of arguments for 'incr' command\r\n"
			}
			return modifyInteger(cmdArgs[1], 1)
			
		case "DECR" :
		if len(cmdArgs) < 2 {
				return "-ERR wrong number of arguments for 'incr' command\r\n"
			}
			return modifyInteger(cmdArgs[1], -1)

		case "LPUSH":
			if len(cmdArgs) < 3 {
				return "-ERR wrong number of arguments for 'lpush' command\r\n"
			}
			return pushList(cmdArgs, true)
		case "RPUSH" :
			if len(cmdArgs) < 3 {
				return "-ERR wrong number of arguments for 'lpush' command\r\n"
			}
			return pushList(cmdArgs, false)
		case "CONFIG":
			return "*-1\r\n"
			
		default:
			return "-ERR unknown command\r\n"
	}
}


const (
	TypeString = "string"
	TypeList = "list"
)

type RedisObject struct {
	Type string
	Value interface{}
	TTL time.Time
}
var (
	data = make(map[string]*RedisObject)
	dataMutex sync.RWMutex
)

func pushList(cmdArgs[] string, isleft bool) (string) {
	key := cmdArgs[1]
	valueArr := cmdArgs[2:]

	dataMutex.Lock()
	defer dataMutex.Unlock()
	obj, exists := data[key]
	
	var subArr[] string
	if exists {
		if obj.Type != TypeList {
			return "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
		}
		subArr = obj.Value.([]string)
	} else {
		obj = &RedisObject{Type: TypeList}
		data[key] = obj
	}
	if isleft {
		for _, value := range valueArr{
			subArr = append([]string{value}, subArr...)
		}
	} else {
		subArr = append(subArr, valueArr...)
	}
	obj.Value = subArr
	return ":" + strconv.Itoa(len(subArr)) + "\r\n"
}

func getData(cmdArgs[] string) (string){
	dataMutex.RLock()
	value, ok := data[cmdArgs[1]]
	dataMutex.RUnlock()
	if !ok {
		return "$-1\r\n"
	}
	if value.Type != TypeString{
		return "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
	}
	strVal := value.Value.(string) 
	return "+" + strconv.Itoa(len(strVal)) + "\r\n" + strVal + "\r\n"
}

func setData(cmdArgs[] string) (string){
	if len(cmdArgs) < 3 {
		return "-ERR wrong number of arguments for 'set' command\r\n"
	}
	key := cmdArgs[1]
	value := cmdArgs[2]
	var tempTTL time.Time
	for i := 3; i < len(cmdArgs); i++ {
		flag := strings.ToUpper(cmdArgs[i])

		if flag == "EX" || flag == "PX" || flag == "EXAT" || flag == "PXAT" {
			if i+1 >= len(cmdArgs){
				return "-ERR syntax error\r\n"
			}
			arg, err := strconv.Atoi(cmdArgs[i+1])
			if err != nil {
				return "-ERR value is not an integer or out of range\r\n"
			}
			switch flag {
				case "EX" :
					tempTTL = time.Now().Add(time.Duration(arg) * time.Second)
				case "PX" :
					tempTTL = time.Now().Add(time.Duration(arg) * time.Millisecond)
				case "EXAT" :
					tempTTL = time.Unix(int64(arg), 0)
				case "PXAT" :
					tempTTL = time.UnixMilli(int64(arg))
			}
			i++
		}
	}
		dataMutex.Lock()
		data[key] = &RedisObject{
			Type : TypeString,
			Value: value,
			TTL: tempTTL,
		}
		dataMutex.Unlock()
		return "+OK\r\n"
}

func modifyInteger(key string, n int) (string){
	dataMutex.Lock()
	defer dataMutex.Unlock()

	var currentInt int
	obj, exists := data[key]
	if exists {
		if obj.Type != TypeString{
			return "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
		}
		var err error
		currentInt, err = strconv.Atoi(obj.Value.(string))
		if err != nil {
			return "-ERR value is not an integer or out of range\r\n"
		}
	} else {
		obj = &RedisObject{
			Type: TypeString,
		}
		data[key] = obj
		currentInt = 0
	}
	currentInt += n

	obj.Value = strconv.Itoa(currentInt)
	return ":" + strconv.Itoa(currentInt) + "\r\n"
}

func checkExpiration(key string) bool {
	obj, exists := data[key]
	
}
