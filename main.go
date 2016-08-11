// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/JustinBeckwith/go-yelp/yelp"
)

var bot *linebot.Client
var richbot *linebot.RichMessageRequest
var o *yelp.AuthOptions

func main() {
	strID := os.Getenv("ChannelID")
	numID, err := strconv.ParseInt(strID, 10, 64)
	if err != nil {
		log.Fatal("Wrong environment setting about ChannelID")
	}

	bot, err = linebot.NewClient(numID, os.Getenv("ChannelSecret"), os.Getenv("MID"))
	log.Println("Bot:", bot, " err:", err)

	// check environment variables
	o = &yelp.AuthOptions{
		ConsumerKey:       os.Getenv("CONSUMER_KEY"),
		ConsumerSecret:    os.Getenv("CONSUMER_SECRET"),
		AccessToken:       os.Getenv("ACCESS_TOKEN"),
		AccessTokenSecret: os.Getenv("ACCESS_TOKEN_SECRET"),
	}

	if o.ConsumerKey == "" || o.ConsumerSecret == "" || o.AccessToken == "" || o.AccessTokenSecret == "" {
		log.Println("Wrong environment setting about yelp-api-keys")
	}

	http.HandleFunc("/callback", callbackHandler)
	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	received, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, result := range received.Results {
		content := result.Content()

		if content != nil && content.IsOperation && content.OpType == 4{
			_, err := bot.SendText([]string{result.RawContent.Params[0]}, "Hi～\n歡迎加入 LINE Delicious！\n請輸入'食物 地區' 查詢想吃的美食\n例如:\n義大利麵 新北市新莊區")
			//_, err = bot.SendSticker([]string{result.RawContent.Params[0]}, 11, 1, 100)
			if err != nil {
				log.Println("New friend add event.")
			}
		}

		if content != nil && content.IsMessage && content.ContentType == linebot.ContentTypeText {
			text, err := content.TextContent()
			c := strings.Split(text.Text, " ")
			// create a new yelp client with the auth keys
			client := yelp.New(o, nil)
			if len(c) == 2{
				// make a simple query
				results, err := client.DoSimpleSearch(c[0], c[1])
				if err != nil {
					log.Println(err)
				}

				for i := 0; i <5; i++ {
					imgurl := results.Businesses[i].ImageURL
					address := strings.Join(results.Businesses[i].Location.DisplayAddress,",")
					var largeImageURL = strings.Replace(results.Businesses[i].ImageURL, "ms.jpg", "l.jpg", 1)
					_, err = bot.SendImage([]string{content.From}, results.Businesses[i].MobileURL, largeImageURL)
					imgurl = "http://i.imgur.com/lVM92n5.jpg"
					bot.NewRichMessage(1040).
						SetAction("food", "food", results.Businesses[i].URL).
						SetListener("food", 0, 0, 1040, 1040).
						Send([]string{content.From}, imgurl, "imagURLtest")
					
					_, err = bot.SendText([]string{content.From}, "店名: " + results.Businesses[i].Name + "\n電話: " + results.Businesses[i].Phone + "\n評比: " + strconv.FormatFloat(float64(results.Businesses[i].Rating), 'f', 1, 64))
					_, err = bot.SendLocation([]string{content.From}, "地址: ", address, float64(results.Businesses[i].Location.Coordinate.Latitude), float64(results.Businesses[i].Location.Coordinate.Longitude))
				}
			}else{
				_, err = bot.NewMultipleMessage().
				AddText("輸入格式錯誤, 請確認").
				//AddSticker(1, 1, 100).
				Send([]string{content.From})
			}
			if err != nil {
				log.Println("OK")
			}

			_, err = bot.SendText([]string{content.From}, "請輸入'食物 地區' 查詢想吃的美食\n例如:\n義大利麵 新北市新莊區")
			//_, err = bot.SendSticker([]string{content.From}, 11, 1, 100)
			if err != nil {
				log.Println("wait for new message")
			}
		}
	}
}
