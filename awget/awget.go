package main

import (
	"os"
	"fmt"
	"bufio"
	"strings"
	"strconv"
	"math/rand"
	"time"
	"net"
	"io"
)

type SS struct {
	ip string
	port int
}

// Class variables
var URL string
var chaingangFile string
var ss []SS
var debug bool

func main() {
	debug = false

	// Check arguments
	checkArgs()

	// Open chaingang file
	openFile()

	// Get random SS
	nextSS := ss[getSSNum()]

	// Print info to console
	printInfo(nextSS)

	// Open socket
	openSocket(nextSS)

	fmt.Println("\n\nawget exiting")
}

func openSocket(nextSS SS) {
	// Create and bind socket
	if debug {fmt.Println(">	Attempting to bind socket [" + nextSS.ip + ":" + strconv.Itoa(nextSS.port) + "]...")}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", nextSS.ip + ":" + strconv.Itoa(nextSS.port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(-1)
	}
	if debug {fmt.Println(">	Socket bound")}

	// Connect to socket
	if debug {fmt.Println(">	Attempting to connect to socket...")}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(-1)
	}
	if debug {fmt.Println("> Socket connected")}

	// Write data to socket
	conn.Write([]byte(URL))
	conn.Write([]byte("\r\n\r\n"))

	for i := 0; i < len(ss); i++ {
		buf := make([]byte, 1024)
		result, _ := conn.Read(buf)
		_ = string(buf[:result])

		ssPort := strconv.Itoa(ss[i].port)
		ssInfo := ss[i].ip + ":" + ssPort
		conn.Write([]byte(ssInfo))
	}
	conn.Write([]byte("\n\r\n\r"))
	fmt.Println("waiting for file...")

	// Read file from socket
	fileReceived := false
	if debug {fmt.Println("> fileRecieved == false. Waiting...")}
	for {
		if fileReceived {break}
		buf := make([]byte, 2048)
		result, _ := conn.Read(buf)
		fileName := string(buf[:result])
		if len(fileName) != 0 {
			fileReceived = true
			fmt.Println("File name [" + string(fileName) + "] received!")

			bufFileSize := make([]byte, 10)
			conn.Read(bufFileSize)
			fileSize, _ := strconv.ParseInt(strings.Trim(string(bufFileSize), ":"), 10, 64)
			if debug {fmt.Printf(">	File size: %d\n", fileSize)}
			newFile, err := os.Create(fileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
				os.Exit(-1)
			}

			var receivedBytes int64
			for {
				if (fileSize - receivedBytes) < 2048 {
					io.CopyN(newFile, conn, (fileSize - receivedBytes))
					conn.Read(make([]byte, (receivedBytes + 2048) - fileSize))
					break
				}
				io.CopyN(newFile, conn, 2048)
				receivedBytes += 2048
			}
			if debug {fmt.Println(">	fileRecieved == true")}
		}
	}

	conn.Close()
}

func printInfo (nextSS SS) {
	fmt.Println("Request: ", URL)
	fmt.Println("chainlist is")
	for i := 0; i < len(ss); i++ {
		fmt.Println(ss[i].ip + " " + strconv.Itoa(ss[i].port))
	}
	fmt.Println("next SS is " + nextSS.ip + ":" + strconv.Itoa(nextSS.port))
}

func getSSNum()  int{

	rand.Seed(time.Now().Unix())
	//fmt.Print("Size: ")
	//fmt.Println(len(ss))
	return rand.Intn(len(ss))
}

func checkArgs() {
	if len(os.Args) == 2 {
		URL = os.Args[1]
		chaingangFile = "chaingang.txt"
	} else if len(os.Args) == 4 {
		URL = os.Args[1]
		chaingangFile = os.Args[3]
	} else {
		fmt.Println("Error: Incorrect number of arguments")
		os.Exit(-1)
	}
}

func openFile() {
	file, err := os.Open(chaingangFile)
	if err != nil {
		fmt.Println("Could not open file!")
		os.Exit(-1)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var line string
	for scanner.Scan() {
		line = scanner.Text()
		if len(line) == 1 {continue}
		//fmt.Println(line)
		stringSlice := strings.Split(line, " ")
		i := stringSlice[0]
		p, _ := strconv.Atoi(stringSlice[1])
		s := SS{i, p}
		ss = append(ss, s)

	}
}