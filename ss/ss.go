package main

import (
	"os"
	"fmt"
	"net"
	"log"
	"strings"
	"strconv"
	"math/rand"
	"time"
	"bufio"
	"os/exec"
	"path/filepath"
	"io"
)

// Class variables
var ip net.IP
var port string
var URL string
var ss []SS
var debug bool
var fileReceived bool

type SS struct {
	ip string
	port int
}

func main() {
	debug = false

	// Check arguments (and get port)
	checkArgsSS()

	// Get IP address
	getIP()

	// Populate address information
	fmt.Printf("ss %v : %v\n", ip, port)

	// Create socket
	createServerSocket()

	fmt.Println("\n\nss exiting")

}

func createClientSocket(nextSS SS) {
	// Create and bind socket
	if debug {fmt.Println(">	Client socket attempting to bind. [" + nextSS.ip + ":" + strconv.Itoa(nextSS.port) + "]...")}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", nextSS.ip + ":" + strconv.Itoa(nextSS.port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(-1)
	}
	if debug {fmt.Println(">	Client socket bound")}

	// Connect to socket
	if debug {fmt.Println(">	Client socket attempting to connect...")}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(-1)
	}
	if debug {fmt.Println(">	Client socket connected")}

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

	// Read file data
	//time.Sleep(time.Second * 5)

	if debug {fmt.Println("> fileRecieved == false. Waiting...")}
	for {
		if fileReceived {break}
		buf := make([]byte, 2048)
		result, _ := conn.Read(buf)
		fileName := string(buf[:result])
		if len(fileName) != 0 {
			bufFileSize := make([]byte, 10)
			conn.Read(bufFileSize)
			fileSize, _ := strconv.ParseInt(strings.Trim(string(bufFileSize), ":"), 10, 64)
			if debug {
				fmt.Print(">	File size: ")
				fmt.Println(fileSize)
			}
			var receivedBytes int64
			fileReceived = true
			newFile, err := os.Create(fileName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
				os.Exit(-1)
			}
			defer newFile.Close()

			for {
				if (fileSize - receivedBytes) < 2048 {
					io.CopyN(newFile, conn, fileSize - receivedBytes)
					conn.Read(make([]byte, (receivedBytes + 2048) - fileSize))
					break
				}
				io.CopyN(newFile, conn, 2048)
				receivedBytes += 2048
			}

			fmt.Println("File [" + string(fileName) + "] downloaded!")
			if debug {fmt.Println(">	fileRecieved == true. Sending file...")}
		}
	}

	conn.Close()
}



func getSSNum()  int{
	rand.Seed(time.Now().Unix())
	return rand.Intn(len(ss))
}

func createServerSocket() {
	fileReceived = false

	// Create and bind socket
	if debug {fmt.Println(">	Server socket attempting to bind...")}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ":" + port)
	if err != nil {
		fmt.Println("Error: Could not resolve TCP address at port " + port)
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(-1)
	}
	if debug {fmt.Println(">	Server socket bound")}

	// Listen on socket
	if debug {fmt.Println(">	Server socket attempting to listen...")}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		fmt.Println("Error: TCP listen failed at port " + port)
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(-1)
	}
	if debug {fmt.Println(">	Server socket listening")}

	for {
		// Accept connection on socket
		if debug {fmt.Println(">	Server socket waiting for connection to accept...")}
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
			os.Exit(-1)
		}
		if debug {fmt.Println(">	Server socket accepted")}
		go handleServerSocket(conn)
		//if debug {time.Sleep(10000 * time.Millisecond)}
	}
}

func handleServerSocket(conn net.Conn) {
	//time.Sleep(5000 * time.Millisecond)
	// Read from client
	defer conn.Close()
	buf := make([]byte, 2048)
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	result, err := r.Read(buf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(-1)
	}
	url := string(buf[:result])
	URL = url[:len(url)-4]
	fmt.Println("Request: " + URL)

	// Get SS info
	for {
		w.Write([]byte(" "))
		w.Flush()
		result, err = r.Read(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
			os.Exit(-1)
		}
		ssInfo := string(buf[:result])
		if strings.HasSuffix(ssInfo, "\n\r\n\r") {
			ssInfo = ssInfo[:len(ssInfo)-4]
			seperateData(ssInfo)
			break
		}
		seperateData(ssInfo)
	}

	fmt.Print("chainlist is ")
	if len(ss) == 0 {
		fmt.Println("empty")
	} else {
		fmt.Println()
	}
	for i := 0; i < len(ss); i++ {
		fmt.Println(ss[i].ip + ":" + strconv.Itoa(ss[i].port))
	}

	// If chainlist isn't empty, go to next SS
	if len(ss) != 0 {
		// Get random SS
		nextSS := ss[getSSNum()]
		fmt.Println("next SS is " + nextSS.ip + ":" + strconv.Itoa(nextSS.port))
		createClientSocket(nextSS) // Clear chainlist
		ss = ss[:0]

	// If SS is the last one in the chainlist, use wget
	} else {
		if debug {
			fmt.Println(">	Chainlist is empty, retrieving file from internet")
		}
		cmd := exec.Command("wget", URL)
		err := cmd.Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
			os.Exit(-1)
		}
		if debug {
			fmt.Println(">	wget [" + filepath.Base(URL) + "] completed succesfully")

		}
		time.Sleep(time.Second * 3)
		fmt.Println("File ["+ filepath.Base(URL) + " ]downloaded! Relaying...")
	}


	// Write file back to client
	if debug {fmt.Println(">	File sending ...")}
	w.Flush()
	fileName := filepath.Base(URL)
	w.Write([]byte(fileName))
	w.Flush()
	if debug {fmt.Println(">	Opening file")}
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(-1)
	}
	if debug{fmt.Println(">	File opened")}
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(-1)
	}
	if debug{fmt.Println(">	File Stats read")}
	fileSize := fillString(strconv.FormatInt(fileInfo.Size(), 10), 10)
	conn.Write([]byte(fileSize))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(-1)
	}
	for {
		_, err = file.Read(buf)
		if err == io.EOF {break}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
			os.Exit(-1)
		}
		conn.Write(buf)
	}
	fmt.Println("File [" + fileName + "] sent! Goodbye!")

	// Close current connection
	if debug {fmt.Println(">	Connection thread closing")}
	conn.Close()
	fmt.Println("Waiting for new requests...")
}

func fillString(retunString string, toLength int) string {
	for {
		lengtString := len(retunString)
		if lengtString < toLength {
			retunString = retunString + ":"
			continue
		}
		break
	}
	return retunString
}

func seperateData(data string) {
	//fmt.Println("Data: " + data)
	addressInfo := strings.Split(data, ":")
	iAddr := addressInfo[0]
	p, _ := strconv.Atoi(addressInfo[1])
	s := SS{iAddr, p}
	// Remove self from chainlist
	if strings.EqualFold(iAddr+":"+addressInfo[1], ip.String()+":"+port) {
		return
	}
	ss = append(ss, s)
}

func getIP() {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	ip = localAddr.IP
}

func checkArgsSS()  {
	if len(os.Args ) == 1 {
		port = "2100"
	}
	if len(os.Args) == 3 {
		port = os.Args[2]
	} else {
		fmt.Println("Error: Incorrect number of arguments")
		os.Exit(-1)
	}
}

