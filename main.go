package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/valyala/fasthttp"
)

var guilds = make(map[string]string)
var socket *websocket.Conn

func main() {
	socketURL := "wss://gateway.discord.gg/?v=10&encoding=json"
	var err error
	socket, _, err = websocket.DefaultDialer.Dial(socketURL, nil)
	if err != nil {
		log.Fatal("Error connecting to WebSocket:", err)
	}
	defer socket.Close()

	go handleMessages()

	for {
		select {}
	}
}

func handleMessages() {
	for {
		_, message, err := socket.ReadMessage()
		if err != nil {
			log.Println("Error reading message from WebSocket:", err)
			return
		}

		var data map[string]interface{}
		err = json.Unmarshal(message, &data)
		if err != nil {
			log.Println("Error decoding JSON:", err)
			continue
		}

		eventType, _ := data["t"].(string)

		switch eventType {
		case "GUILD_UPDATE", "GUILD_DELETE":
			guildID := data["d"].(map[string]interface{})["guild_id"].(string)
			guild, ok := guilds[guildID]
			if ok {
				patchURL := "https://discordapp.com/api/v8/guilds/GUILDID/vanity-url"
				postURL := "https://discordapp.com/api/v8/channels/CHANNELID/messages"
				patchData := map[string]string{"code": guild}

				patchResponse, err := patchDataToDiscordAPI(patchURL, patchData)
				if err != nil {
					log.Printf("Error patching data: %v\n", err)
					continue
				}

				content := fmt.Sprintf("%s | Checked: https://discord.gg/%s | ", eventType, guild)
				if patchResponse {
					content = fmt.Sprintf("%s | Vanity: %s @everyone| ", eventType, guild)
				}

				postData := map[string]string{"content": content}
				_, err = postDataToDiscordAPI(postURL, postData)
				if err != nil {
					log.Printf("Error posting data: %v\n", err)
					continue
				}

				delete(guilds, guildID)
			}
		case "READY":
			guildList := data["d"].(map[string]interface{})["guilds"].([]interface{})
			for _, guild := range guildList {
				guildMap := guild.(map[string]interface{})
				if vanityURLCode, exists := guildMap["vanity_url_code"].(string); exists {
					guilds[guildMap["id"].(string)] = vanityURLCode
				}
			}
		}

		if opCode, exists := data["op"].(float64); exists && opCode == 10 {
			token := "LİSTENERTOKEN"
			intents := 1 << 0
			properties := map[string]string{"os": "Windows", "browser": "Chrome", "device": "Canary"}
			authData := map[string]interface{}{"token": token, "intents": intents, "properties": properties}
			socket.WriteJSON(map[string]interface{}{"op": 2, "d": authData})

			heartbeatInterval := int(data["d"].(map[string]interface{})["heartbeat_interval"].(float64))
			go func() {
				for range time.Tick(time.Duration(heartbeatInterval) * time.Millisecond) {
					socket.WriteJSON(map[string]interface{}{"op": 1, "d": map[string]interface{}{}})
				}
			}()
		} else if opCode, exists := data["op"].(float64); exists && opCode == 7 {
			log.Println(data)
			log.Println("Received opcode 7. Exiting...")
			socket.Close()
			return
		}
	}
}

func patchDataToDiscordAPI(url string, data map[string]string) (bool, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, err
	}

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(url)
	req.Header.SetMethod("PATCH")
	req.Header.Set("Authorization", "SNİPTOKEN")
	req.Header.Set("Content-Type", "application/json")
	req.SetBody(jsonData)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	client := &fasthttp.Client{}
	err = client.Do(req, resp)
	if err != nil {
		return false, err
	}

	return resp.StatusCode() == fasthttp.StatusOK, nil
}

func postDataToDiscordAPI(url string, data map[string]string) (bool, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, err
	}

	req := fasthttp.AcquireRequest()
	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.Set("Authorization", "SNİPTOKEN")
	req.Header.Set("Content-Type", "application/json")
	req.SetBody(jsonData)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	client := &fasthttp.Client{}
	err = client.Do(req, resp)
	if err != nil {
		return false, err
	}

	return resp.StatusCode() == fasthttp.StatusOK, nil
}
