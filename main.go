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
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/JustinBeckwith/go-yelp/yelp"
	"github.com/guregu/null"
	"github.com/line/line-bot-sdk-go/linebot"
)

var bot *linebot.Client
var o *yelp.AuthOptions
var food = make(map[string]string)

type UrlShortener struct {
	ShortUrl    string
	OriginalUrl string
}

func main() {
	rand.Seed(time.Now().UnixNano())
	strID := os.Getenv("ChannelID")
	numID, err := strconv.ParseInt(strID, 10, 64)
	if err != nil {
		log.Fatal("Wrong environment setting about ChannelID")
	}

	bot, err = linebot.NewClient(numID, os.Getenv("ChannelSecret"), os.Getenv("MID"))
	if err != nil {
		log.Fatal("Wrong environment setting about ChannelSecret and MID")
	}

	// check environment variables
	o = &yelp.AuthOptions{
		ConsumerKey:       os.Getenv("CONSUMER_KEY"),
		ConsumerSecret:    os.Getenv("CONSUMER_SECRET"),
		AccessToken:       os.Getenv("ACCESS_TOKEN"),
		AccessTokenSecret: os.Getenv("ACCESS_TOKEN_SECRET"),
	}

	if o.ConsumerKey == "" || o.ConsumerSecret == "" || o.AccessToken == "" || o.AccessTokenSecret == "" {
		log.Fatal("Wrong environment setting about yelp-api-keys")
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

	// create a new yelp client with the auth keys
	client := yelp.New(o, nil)

	for _, result := range received.Results {
		content := result.Content()

		//identify different ContentType
		if content != nil && content.IsOperation && content.OpType == 4 {
			//add new friend
			_, err := bot.SendText([]string{result.RawContent.Params[0]}, "Hi~\n歡迎加入 Delicious!\n\n想查詢附近或各地美食都可以LINE我呦！\n\n請問你想吃什麼?\nex:義大利麵\n\n想不到吃什麼，也可以直接'傳送目前位置訊息'")
			var img = "http://imageshack.com/a/img921/318/DC21al.png"
			_, err = bot.SendImage([]string{content.From}, img, img)
			if err != nil {
				log.Println(err)
			}
		} else if content != nil && content.ContentType == linebot.ContentTypeLocation {
			//receive location
			loc, err := content.LocationContent()
			if err != nil {
				log.Println(err)
			}

			if food[content.From] == "" {
				//_, err = bot.SendText([]string{content.From},"想不到吃什麼，也可以直接'傳送目前位置訊息'")
				food[content.From] = "food,restaurants"
			}

			// Build an advanced set of search criteria that include
			// general options, and coordinate options.
			s := yelp.SearchOptions{
				GeneralOptions: &yelp.GeneralOptions{
					Term: food[content.From],
				},
				CoordinateOptions: &yelp.CoordinateOptions{
					Latitude:  null.FloatFrom(loc.Latitude),
					Longitude: null.FloatFrom(loc.Longitude),
				},
			}

			// Perform the search using the search options
			results, err := client.DoSearch(s)
			if err != nil {
				log.Println(err)
				_, err = bot.SendText([]string{content.From}, "查無資料！\n請重新輸入\n\n請問你想吃什麼?\nex:義大利麵\n\n想不到吃什麼，也可以直接'傳送目前位置訊息'\nex：")
				var img = "http://imageshack.com/a/img921/318/DC21al.png"
				_, err = bot.SendImage([]string{content.From}, img, img)
				delete(food, content.From)
			}

			for j := 0; j < 3; j++ {
				i := 0
				if results.Total >= 16 {
					i = rand.Intn(16)
				} else if results.Total >= 8 {
					i = rand.Intn(8)
				} else if results.Total > j {
					i = j
				} else if results.Total <= j && results.Total != 0 {
					_, err = bot.SendText([]string{content.From}, "已無更多資料！")
					break
				}
				urlOrig := UrlShortener{}
				urlOrig.short(results.Businesses[i].MobileURL)
				address := strings.Join(results.Businesses[i].Location.DisplayAddress, ",")
				var largeImageURL = strings.Replace(results.Businesses[i].ImageURL, "ms.jpg", "l.jpg", 1)

				_, err = bot.SendImage([]string{content.From}, largeImageURL, largeImageURL)
				_, err = bot.SendText([]string{content.From}, "店名："+results.Businesses[i].Name+"\n電話："+results.Businesses[i].Phone+"\n評比："+strconv.FormatFloat(float64(results.Businesses[i].Rating), 'f', 1, 64)+"\n更多資訊："+urlOrig.ShortUrl)
				_, err = bot.SendLocation([]string{content.From}, results.Businesses[i].Name+"\n", address, float64(results.Businesses[i].Location.Coordinate.Latitude), float64(results.Businesses[i].Location.Coordinate.Longitude))
			}
			_, err = bot.SendText([]string{content.From}, "請問你想吃什麼?\nex:義大利麵\n\n想不到吃什麼，也可以直接'傳送目前位置訊息'\nex：")
			var img = "http://imageshack.com/a/img921/318/DC21al.png"
			_, err = bot.SendImage([]string{content.From}, img, img)
			delete(food, content.From)
		} else if content != nil && content.IsMessage && content.ContentType == linebot.ContentTypeText {
			//receive text
			text, err := content.TextContent()
			if err != nil {
				log.Println(err)
			}
			log.Println("food: " + food[content.From])
			if food[content.From] == "" {
				food[content.From] = text.Text
				_, err := bot.SendText([]string{content.From}, "你在哪裡?\n請'手動輸入目前位置'\nex:台北市信義區...\n或是利用'傳送目前位置訊息'\nex：")
				var img = "http://imageshack.com/a/img921/318/DC21al.png"
				_, err = bot.SendImage([]string{content.From}, img, img)
				if err != nil {
					log.Println(err)
				}
			} else {
				// make a simple query for food and location
				results, err := client.DoSimpleSearch(food[content.From], text.Text)
				if err != nil {
					log.Println(err)
					_, err = bot.SendText([]string{content.From}, "查無資料！\n請重新輸入\n\n請問你想吃什麼?\nex:義大利麵\n\n不知道吃什麼\n可以直接'傳送目前位置訊息'\nex:")
					var img = "http://imageshack.com/a/img921/318/DC21al.png"
					_, err = bot.SendImage([]string{content.From}, img, img)
					delete(food, content.From)
				}

				for j := 0; j < 3; j++ {
					i := 0
					if results.Total >= 20 {
						i = rand.Intn(20)
					} else if results.Total >= 10 {
						i = rand.Intn(10)
					} else if results.Total > j {
						i = j
					} else if results.Total <= j && results.Total != 0 {
						_, err = bot.SendText([]string{content.From}, "已無更多資料！")
						break
					}
					urlOrig := UrlShortener{}
					urlOrig.short(results.Businesses[i].MobileURL)
					address := strings.Join(results.Businesses[i].Location.DisplayAddress, ",")
					var largeImageURL = strings.Replace(results.Businesses[i].ImageURL, "ms.jpg", "l.jpg", 1)

					_, err = bot.SendImage([]string{content.From}, largeImageURL, largeImageURL)
					_, err = bot.SendText([]string{content.From}, "店名："+results.Businesses[i].Name+"\n電話："+results.Businesses[i].Phone+"\n評比："+strconv.FormatFloat(float64(results.Businesses[i].Rating), 'f', 1, 64)+"\n更多資訊："+urlOrig.ShortUrl)
					_, err = bot.SendLocation([]string{content.From}, results.Businesses[i].Name+"\n", address, float64(results.Businesses[i].Location.Coordinate.Latitude), float64(results.Businesses[i].Location.Coordinate.Longitude))
				}
				_, err = bot.SendText([]string{content.From}, "請問你想吃什麼?\nex:義大利麵\n\n不知道吃什麼\n可以直接'傳送目前位置訊息'\nex:")
				var img = "http://imageshack.com/a/img921/318/DC21al.png"
				_, err = bot.SendImage([]string{content.From}, img, img)
				delete(food, content.From)
			}
		}
	}
}

func getResponseData(urlOrig string) string {
	response, err := http.Get(urlOrig)
	if err != nil {
		log.Println(err)
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	return string(contents)
}

func isGdShortener(urlOrig string) (string, string) {
	escapedUrl := url.QueryEscape(urlOrig)
	isGdUrl := fmt.Sprintf("http://is.gd/create.php?url=%s&format=simple", escapedUrl)
	return getResponseData(isGdUrl), urlOrig
}

func (u *UrlShortener) short(urlOrig string) *UrlShortener {
	shortUrl, originalUrl := isGdShortener(urlOrig)
	u.ShortUrl = shortUrl
	u.OriginalUrl = originalUrl
	return u
}
