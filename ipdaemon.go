package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const DEBUG = false

func main() {

	conf := parseConfigFile()

	ip, err := externalIP()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	fmt.Println("Local IP:", ip)

	if DEBUG {
		fmt.Println("API CALL:", conf["notify_api"]+ip)
	}

	for {
		getUrlData(conf["notify_api"] + ip)
		interval, _ := strconv.Atoi(conf["interval"])
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func readLine(r *bufio.Reader) (string, error) {
	var (
		isPrefix bool  = true
		err      error = nil
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

func getUrlData(url string) string {

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("X-Requested-With", "Go IP daemon")

	resp, err := client.Do(req)
	if err != nil {
		//panic(err)
		if DEBUG {
			fmt.Println("Could not notify remote API")
		}
		return ""
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return string(body)
}

func parseConfigFile() map[string]string {

	f, err := os.Open("ipdaemon.conf")
	if err != nil {
		fmt.Printf("Error! Could not open config file: %v\n", err)
		fmt.Println("")
		os.Exit(0)
	}
	defer f.Close()

	r := bufio.NewReader(f)

	params := map[string]string{}

	for err == nil {
		s, err := readLine(r)
		if err != nil {
			break
		}
		if err == nil && s != "" {
			parts := strings.SplitN(s, "=", 2)
			params[parts[0]] = strings.Trim(parts[1], " ")
		}
	}

	return params
}

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("No connected network interfaces found.")
}
