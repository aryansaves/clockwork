package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	loadDatabase()
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
			log.Println("err ", err)
		}
		if len(line) > 0 && line[0] == '*' {
			args, err := strconv.Atoi(strings.TrimSpace(line[1:]))
			if err != nil {
				log.Printf("atoi line error %s",err)
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
			return "$" + strconv.Itoa(len(cmdArgs[1])) + "\r\n" + cmdArgs[1] + "\r\n"
			
		case "EXISTS":
			if len(cmdArgs) < 2 {
				return "-ERR wrong number of arguements for 'exists' command\r\n"
			}
			count := 0
			dataMutex.Lock()
			for _, key := range cmdArgs[1:]{
				if checkExpiration(key) {
					count++
				}
			}
			dataMutex.Unlock()
			return ":" + strconv.Itoa(count) + "\r\n"
		case "DEL" :
			if len(cmdArgs) < 2 {
				return "-ERR wrong number of arguments for 'del' command\r\n"
			}
			count := 0
			dataMutex.Lock()
			for _, key := range cmdArgs[1:]{
				if checkExpiration(key) {
					delete(data, key)
					count++
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
				return "-ERR wrong number of arguments for 'decr' command\r\n"
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

			case "LRANGE":
    	if len(cmdArgs) < 4 {
        	return "-ERR wrong number of arguments for 'lrange' command\r\n"
     	}
      	start, err1 := strconv.Atoi(cmdArgs[2])
       	end, err2 := strconv.Atoi(cmdArgs[3])
        if err1 != nil || err2 != nil {
        	return "-ERR value is not an integer or out of range\r\n"
        }
        dataMutex.RLock()
        defer dataMutex.RUnlock()
        obj, exists := data[cmdArgs[1]]
        if !exists {
        	return "*0\r\n"
        }
        if obj.Type != TypeList {
        	return "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
        }
        list := obj.Value.([]string)
        n := len(list)
        if start < 0 { start = n + start }
        if end < 0 { end = n + end }
        if start < 0 { start = 0 }
        if end >= n { end = n - 1 }
        if start > end {
        	return "*0\r\n"
        }
        var sb strings.Builder
        sb.WriteString("*" + strconv.Itoa(end-start+1) + "\r\n")
        for i := start; i <= end; i++ {
        	v := list[n-1-i] 
         	sb.WriteString("$" + strconv.Itoa(len(v)) + "\r\n" + v + "\r\n")
        }
        return sb.String()
    
		case "CONFIG":
			return "*0\r\n"

		case "SAVE":
			return saveDatabase()
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

	checkExpiration(key)
	
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
    	subArr = append(subArr, valueArr...)  
	} else {
    	reversed := make([]string, len(valueArr))
     	for i, v := range valueArr {
        	reversed[len(valueArr)-1-i] = v
      	}
       subArr = append(reversed, subArr...) 
	}
	obj.Value = subArr
	return ":" + strconv.Itoa(len(subArr)) + "\r\n"
}

func getData(cmdArgs[] string) (string){
	dataMutex.Lock()
	defer dataMutex.Unlock()

	if !checkExpiration(cmdArgs[1]){
		return "$-1\r\n"
	}
	obj := data[cmdArgs[1]]
	if obj.Type != TypeString {
		return "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
	}
	strVal := obj.Value.(string)
	
	return "$" + strconv.Itoa(len(strVal)) + "\r\n" + strVal + "\r\n"
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

	checkExpiration(key)
	
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
	if !exists {
		return false
	}
	if !obj.TTL.IsZero() && time.Now().After(obj.TTL){
		delete(data, key)
		return false
	}
	return true
}

func saveDatabase() (string){
	dataMutex.Lock()
	defer dataMutex.Unlock()

	for key := range data {
		checkExpiration(key)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "-ERR failed to serialize database state\r\n"
	}
	
	tmp := "dump.json.tmp"
    err = os.WriteFile(tmp, jsonData, 0644)
    if err != nil {
        return "-ERR failed to write database file to disk\r\n"
    }
    
    if err := os.Rename(tmp, "dump.json"); err != nil {
        return "-ERR failed to finalize database file\r\n"
    }
    return "+OK\r\n"
}

func loadDatabase() {
	file, err := os.ReadFile("dump.json")
	if err != nil {
		if os.IsNotExist(err) {
			return 
		}
		log.Println("Error reading persistence file:", err)
		return
	}

	dataMutex.Lock()
	defer dataMutex.Unlock()

	var rawData map[string]*RedisObject
	if err := json.Unmarshal(file, &rawData); err != nil {
		log.Println("Error parsing persistence file structural layout:", err)
		return
	}

	for key, obj := range rawData {
		if !obj.TTL.IsZero() && time.Now().After(obj.TTL) {
			continue
		}

		switch obj.Type {
		case TypeString:
			if strVal, ok := obj.Value.(string); ok {
				obj.Value = strVal
			}

		case TypeList:
			if interfaceSlice, ok := obj.Value.([]interface{}); ok {
				stringSlice := make([]string, len(interfaceSlice))
				for i, v := range interfaceSlice {
					stringSlice[i] = fmt.Sprintf("%v", v)
				}
				obj.Value = stringSlice
			}
		}

		data[key] = obj
	}
	log.Printf("Successfully restored %d keys from persistence file.\n", len(data))
}
