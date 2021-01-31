package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	apiHost  = "api.minecraftservices.com"
	authhost = "authserver.mojang.com"
	connPort = ":443"
	connType = "tcp"
)

type NxAPIRes struct {
	DropTime string `json:"drop_time"`
}

type SecurityRes struct {
	Answer AnswerRes `json:"answer"`
}

type AnswerRes struct {
	ID int `json:"id"`
}

type Config struct {
	Name                string `json:"name"`
	Delay               int    `json:"delay"`
	SpeedCap            int    `json:"speedLimit"`
	SnipeReqs           int    `json:"snipeReqs"`
	UseMicrosoftAccount bool   `json:"useMS"` //unused
}

var timestamp time.Time
var name string
var delay int
var sniped bool
var speedcap int
var snipereqs int

func main() {
	sniped = false
	var accts []string
	fmt.Println("GoSnipe//overestimate")
	file, err := os.Open("./accounts.txt")
	if err != nil {
		fmt.Println("Failed to load accounts.txt.\nMake sure the file exists and try again.\nIf the file exists, check the permissions and try again.")
		file.Close()
		os.Exit(2)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		acct := scanner.Text()
		if strings.Contains(acct, "\n") {
			acct = acct[:len(acct)-1]
		}
		if strings.Contains(acct, "\r") {
			acct = acct[:len(acct)-1]
		}
		accts = append(accts, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Failed to parse accounts.txt.\nMake sure the end-of-line format is matches your platform.")
		os.Exit(4)
	}
	var configuration Config
	file2, err := os.Open("./config.json")
	reader := bufio.NewScanner(file2)
	var data []byte
	for reader.Scan() {
		data = append(data, reader.Bytes()...)
	}
	json.Unmarshal(data, &configuration)
	name = configuration.Name
	delay = configuration.Delay
	speedcap = configuration.SpeedCap
	snipereqs = configuration.SnipeReqs
	res, err := http.Get("https://api.nathan.cx/check/" + name)
	if err != nil {
		fmt.Println("failed to connect to droptime server. most likely causes are dead internet and/or the server is down.")
		os.Exit(5)
	}
	apiRes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("failed to parse drop time.")
		os.Exit(6)
	}
	var nxres NxAPIRes
	res.Body.Close()
	json.Unmarshal(apiRes, &nxres)
	timestamp, err = time.Parse(time.RFC3339, nxres.DropTime)
	if err != nil {
		fmt.Println("failed to parse drop time.")
		os.Exit(5)
	}
	for i := 0; i < len(accts); i++ {
		go snipeSetup(accts[i], i)
	}
	go checkFailure()
	fmt.Println("Locked and loaded. Press enter to quit.")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	os.Exit(127) //user terminated
}
func checkFailure() {
	time.Sleep(time.Until(timestamp.Add(time.Millisecond * time.Duration(10000))))
	if sniped == false {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
func snipeSetup(acct string, i int) {
	time.Sleep(time.Until(timestamp.Add(time.Second * time.Duration(-35))))
	conn, err := tls.Dial(connType, authhost+connPort, nil)
	if err != nil {
		fmt.Println("failed to connect to auth server.\nif you see this error once per account, your internet is dead.")
		return
	}
	dataSplit := strings.Split(acct, ":")
	if (len(dataSplit)) != 5 {
		return
	}
	payload := "{\"username\": \"" + dataSplit[0] + "\", \"password\": \"" + dataSplit[1] + "\", \"agent\": {\"name\": \"Minecraft\", \"version\": 1}}"
	data := "POST /authenticate HTTP/1.1\r\nContent-Type: application/json\r\nHost: authserver.mojang.com\r\nUser-Agent: GoSnipe/1.0 golang/unknown\r\nContent-Length: " + strconv.Itoa(len(payload)) + "\r\n\r\n" + payload
	var authbytes []byte
	authbytes = make([]byte, 4096)
	auth := make(map[string]interface{})
	var security []SecurityRes
	conn.Write([]byte(data))
	conn.Read(authbytes)
	conn.Close()
	authbytes = []byte(strings.Split(strings.Split(string(authbytes), "\x00")[0], "\r\n\r\n")[1])
	err = json.Unmarshal(authbytes, &auth)
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.mojang.com/user/security/challenges", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	if auth["accessToken"] == nil {
		fmt.Println("empty token in json")
		return
	}
	req.Header.Set("Authorization", "Bearer "+auth["accessToken"].(string))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	securitybytes, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	err = json.Unmarshal(securitybytes, &security)
	if err != nil {
		fmt.Println(err)
		return
	}
	data = `[{"id": ` + strconv.Itoa(security[0].Answer.ID) + `, "answer": "` + dataSplit[2] + `"}, {"id": ` + strconv.Itoa(security[1].Answer.ID) + `, "answer": "` + dataSplit[3] + `"}, {"id": ` + strconv.Itoa(security[2].Answer.ID) + `, "answer": "` + dataSplit[4] + `"}]`
	b := bytes.NewReader([]byte(data))
	req, err = http.NewRequest("POST", "https://api.mojang.com/user/security/location", b)
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+auth["accessToken"].(string))
	_, _ = client.Do(req)
	for j := 0; j < snipereqs; j++ {
		ch := make(chan int)
		go snipe(auth["accessToken"].(string), dataSplit[0], i*j, ch)
	}
}
func getSnipeRes(ch chan int, s *tls.Conn, email string) {
	var res []byte
	res = make([]byte, 4096)
	var rescodei int
	var rescodes string
	<-ch
	_, err := s.Read(res)
	if err != nil {
		fmt.Println(err)
		return
	}
	rescodes = string(res[9:12])
	rescodei, _ = strconv.Atoi(string(res[9:12]))
	if rescodei == 200 {
		fmt.Println("200 >> Sniped " + name + " on email " + email + " at " + time.Now().Format("2006/01/02 15:04:05.0000000"))
		sniped = true
	} else {
		fmt.Println(rescodes + " >> Failure at time " + time.Now().Format("2006/01/02 15:04:05.0000000"))
	}
	s.Close()
}
func snipe(bearer, email string, i int, ch chan int) {
	time.Sleep(time.Until(timestamp.Add(time.Millisecond * time.Duration(0-10000-delay))))
	conn, err := tls.Dial(connType, apiHost+connPort, nil)
	payload := "PUT /minecraft/profile/name/" + name + " HTTP/1.1\r\nHost: api.minecraftservices.com\r\nAuthorization: Bearer " + bearer + "\r\n"
	if err != nil {
		fmt.Println("failed to open conn")
		return
	}
	conn.Write([]byte(payload))
	go getSnipeRes(ch, conn, email)
	time.Sleep(time.Until(timestamp.Add(time.Millisecond * time.Duration(-delay+i*speedcap))))
	conn.Write([]byte("\r\n"))
	ch <- 0
	fmt.Println("Sent request at " + time.Now().Format("2006/01/02 15:04:05.0000000"))
	return
}
