package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Frame struct {
	Cmd    string   `json:"cmd"`
	Sender string   `json:"sender"`
	Data   []string `json:"data"`
}

type Info struct {
	nextNode string
	nextNum  int
	imFirst  bool
	cont     int
}

var (
	host         string
	remotes      []string
	myNum        int
	chInfo       chan Info
	readyToStart chan bool
)

func main() {
	rand.Seed(time.Now().UnixNano())
	if len(os.Args) == 1 {
		log.Println("Hostname not given")
	} else {
		host = os.Args[1]
		if len(os.Args) >= 3 {
			connectToNode(os.Args[2])
		}
		if len(os.Args) == 4 {
			go func() {
				time.Sleep(5 * time.Second)
				for _, remote := range remotes {
					send(remote, Frame{"agrawalla", host, []string{}}, nil)
				}
				handleAgrawalla()
			}()
		}
		chInfo = make(chan Info)
		readyToStart = make(chan bool)
		server()
	}
}

func connectToNode(remote string) {
	remotes = append(remotes, remote)
	if !send(remote, Frame{"hello", host, []string{}}, func(cn net.Conn) {
		dec := json.NewDecoder(cn)
		var frame Frame
		dec.Decode(&frame)
		remotes = append(remotes, frame.Data...)
		log.Printf("%s: friends: %s\n", host, remotes)
	}) {
		log.Printf("%s: unable to connect to %s\n", host, remote)
	}
}

func send(remote string, frame Frame, callback func(net.Conn)) bool {
	if cn, err := net.Dial("tcp", remote); err == nil {
		defer cn.Close()
		enc := json.NewEncoder(cn)
		enc.Encode(frame)
		if callback != nil {
			callback(cn)
		}
		return true
	} else {
		log.Printf("%s: can't connect to %s\n", host, remote)
		idx := -1
		for i, rem := range remotes {
			if remote == rem {
				idx = i
				break
			}
		}
		if idx >= 0 {
			remotes[idx] = remotes[len(remotes)-1]
			remotes = remotes[:len(remotes)-1]
		}
		return false
	}
}

func server() {
	if ln, err := net.Listen("tcp", host); err == nil {
		defer ln.Close()
		log.Printf("Listening on %s\n", host)
		for {
			if cn, err := ln.Accept(); err == nil {
				go fauxDispatcher(cn)
			} else {
				log.Printf("%s: cant accept connection.\n", host)
			}
		}
	} else {
		log.Printf("Can't listen on %s\n", host)
	}
}

func fauxDispatcher(cn net.Conn) {
	defer cn.Close()
	dec := json.NewDecoder(cn)
	frame := &Frame{}
	dec.Decode(frame)
	switch frame.Cmd {
	case "hello":
		handleHello(cn, frame)
	case "add":
		handleAdd(frame)
	case "agrawalla":
		handleAgrawalla()
	case "num":
		handleNum(frame)
	case "start":
		handleStart()
	}
}

func handleHello(cn net.Conn, frame *Frame) {
	enc := json.NewEncoder(cn)
	enc.Encode(Frame{"<response>", host, remotes})
	notification := Frame{"add", host, []string{frame.Sender}}
	for _, remote := range remotes {
		send(remote, notification, nil)
	}
	remotes = append(remotes, frame.Sender)
	log.Printf("%s: friends: %s\n", host, remotes)
}
func handleAdd(frame *Frame) {
	remotes = append(remotes, frame.Data...)
	log.Printf("%s: friends: %s\n", host, remotes)
}
func handleAgrawalla() {
	response, err := http.Get("http://localhost:8000/")
	jsondata := "nothing"
	if err != nil {
		fmt.Printf("The Http request failed with error %s\n", err)
	} else {
		data, _ := ioutil.ReadAll(response.Body)
		fmt.Println(string(data))
		jsondata = string(data)
	}
	myNum := len(jsondata)
	log.Printf("%s: my number is %d\n", host, myNum)
	msg := Frame{"num", host, []string{strconv.Itoa(myNum)}}
	for _, remote := range remotes {
		send(remote, msg, nil)
	}
	chInfo <- Info{"", 1000000001, true, 0}
}
func handleNum(frame *Frame) {
	if num, err := strconv.Atoi(frame.Data[0]); err == nil {
		info := <-chInfo
		if num > myNum {
			if num < info.nextNum {
				info.nextNum = num
				info.nextNode = frame.Sender
			}
		} else {
			info.imFirst = false
		}
		info.cont++
		go func() { chInfo <- info }()
		if info.cont == len(remotes) {
			if info.imFirst {
				log.Printf("%s: I'm first!\n", host)
				criticalSection()
			} else {
				readyToStart <- true
			}
		}
	} else {
		log.Printf("%s: can't convert %v\n", host, frame)
	}
}
func handleStart() {
	<-readyToStart
	criticalSection()
}

func criticalSection() {
	log.Printf("%s: my time has come!\n", host)
	info := <-chInfo
	if info.nextNode != "" {
		log.Printf("%s: letting %s start\n", host, info.nextNode)
		send(info.nextNode, Frame{"start", host, []string{}}, nil)
	} else {
		log.Printf("%s: I was the last one :(\n", host)
	}
}
