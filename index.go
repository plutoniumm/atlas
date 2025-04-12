package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sync"

	"google.golang.org/protobuf/proto"
	pb "manav.ch/atlas/proto"
)

type JSN = map[string]interface{}

func AQuery(bssid string) JSN {
	dataBSSID := fmt.Sprintf("\x12\x13\n\x11%s\x18\x00\x20\x01", bssid)
	payload := fmt.Sprintf("\x00\x01\x00\x05en_US\x00\x13com.apple.locationd\x00\x0a8.1.12B411\x00\x00\x00\x01\x00\x00\x00%c%s",
		len(dataBSSID), dataBSSID)

	resp, err := http.Post("https://gs-loc.apple.com/clls/wloc",
		"application/x-www-form-urlencoded",
		bytes.NewReader([]byte(payload)),
	)

	if err != nil {
		return JSN{"module": "apple", "error": err.Error()}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return JSN{"module": "apple", "error": err.Error()}
	}

	var result pb.BSSIDResp
	if err := proto.Unmarshal(body[10:], &result); err != nil {
		return JSN{"module": "apple", "error": err.Error()}
	}

	if len(result.Wifi) > 0 {
		loc := result.Wifi[0]
		lat := float64(loc.Location.Lat) / 1e8
		lon := float64(loc.Location.Lon) / 1e8
		return JSN{
			"module":    "apple",
			"bssid":     bssid,
			"latitude":  lat,
			"longitude": lon,
		}
	}

	return JSN{"module": "apple", "error": "no location data"}
}

func GQuery(bssid string) JSN {
	apiKey := "" // <-- Insert your API key here
	if apiKey == "" {
		return JSN{"module": "google", "error": "API key is required"}
	}

	url := fmt.Sprintf("https://www.googleapis.com/geolocation/v1/geolocate?key=%s", apiKey)

	payload := JSN{
		"considerIp": false,
		"wifiAccessPoints": []map[string]string{
			{"macAddress": bssid},
			{"macAddress": "00:25:9c:cf:1c:ad"},
		},
	}

	data, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return JSN{"module": "google", "error": err.Error()}
	}
	defer resp.Body.Close()

	var result struct {
		Location struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"location"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return JSN{"module": "google", "error": err.Error()}
	}

	if result.Error.Message != "" {
		return JSN{"module": "google", "error": result.Error.Message}
	}

	return JSN{
		"module":    "google",
		"bssid":     bssid,
		"latitude":  result.Location.Lat,
		"longitude": result.Location.Lng,
	}
}

func isValidBSSID(bssid string) bool {
	regex := regexp.MustCompile(`(?i)^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`)
	return regex.MatchString(bssid)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("❌ Error: No BSSID provided")
		return
	}
	bssid := os.Args[1]

	if !isValidBSSID(bssid) {
		fmt.Println("❌ Error: Invalid BSSID format")
		return
	}

	var wg sync.WaitGroup
	results := make(chan JSN, 2)

	wg.Add(2)

	go func() {
		defer wg.Done()
		results <- AQuery(bssid)
	}()

	go func() {
		defer wg.Done()
		results <- GQuery(bssid)
	}()

	wg.Wait()
	close(results)

	for res := range results {
		if err, ok := res["error"]; ok {
			fmt.Printf("🔴 %s error: %v\n", res["module"], err)
		} else {
			fmt.Printf("🟢 %s result: BSSID=%v, Lat=%v, Lon=%v\n",
				res["module"], res["bssid"], res["latitude"], res["longitude"])
		}
	}
}
