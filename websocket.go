package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Home")
}

func handler(conn *websocket.Conn) {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			weatherData := getWeatherData()
			issData := getISSData()
			splitWeather := strings.Split(weatherData, "Last Updated:")
			type data struct {
				Weather string
				Updated string
				ISS     string
			}
			d := data{
				Weather: splitWeather[0],
				Updated: splitWeather[1],
				ISS:     issData,
			}
			dByte, err := json.Marshal(d)
			if err != nil {
				log.Println("error marshaling data:", err)
			}
			if err := conn.WriteMessage(websocket.TextMessage, dByte); err != nil {
				log.Println("write:", err)
			}
		}
	}()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("Received message:", string(p))

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println("Error writing message:", err)
			return
		}
	}
}

func getWeatherData() string {
	key := "Insert API key here"
	url := "http://api.weatherapi.com/v1/current.json?key=" + key + "&q=Santa-Clara,California&aqi=no"
	response, err := http.Get(url)
	if err != nil {
		log.Println("Error making HTTP request:", err)
		return ""
	}
	if response.StatusCode >= 400 {
		log.Println("Error: API returned a", response.StatusCode, "status code")
		return ""
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println("Error reading response body:", err)
		return ""
	}
	if len(data) == 0 {
		log.Println("Error: response body is empty")
		return ""
	}
	var result map[string]interface{}
	json.Unmarshal([]byte(data), &result)
	tempF := result["current"].(map[string]interface{})["temp_f"]

	localTime := result["location"].(map[string]interface{})["localtime"]
	return fmt.Sprintf("Temperature: %v Last Updated: %v", tempF, localTime)
}

func getISSData() string {
	url := "http://api.open-notify.org/iss-now.json"
	response, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
	}
	var result map[string]interface{}
	json.Unmarshal([]byte(data), &result)
	latitude := result["iss_position"].(map[string]interface{})["latitude"]
	longitude := result["iss_position"].(map[string]interface{})["longitude"]
	return fmt.Sprintf("ISS Latitude: %v, ISS Longitude: %v", latitude, longitude)
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading WebSocket connection:", err)
		return
	}
	fmt.Println("Client Successfully Connected")
	handler(ws)
}

func routes() {
	http.HandleFunc("/", home)
	http.HandleFunc("/ws", wsEndpoint)
}

func main() {
	routes()
	http.ListenAndServe(":8080", nil)
}