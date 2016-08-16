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
	"net/url"
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/JustinBeckwith/go-yelp/yelp"
	"github.com/guregu/null"
	"github.com/line/line-bot-sdk-go/linebot"
)

var bot *linebot.Client
var o *yelp.AuthOptions

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

		//identify different ContentType
		if content != nil && content.IsOperation && content.OpType == 4 {
			//add new friend
			_, err := bot.SendText([]string{result.RawContent.Params[0]}, "Hi~\n歡迎加入 Delicious!\n如果想查詢附近或各地美食都可以LINE我呦！\n\n有下列3種輸入方式：\n1.地區\nex：台北市信義區\n\n2.食物 地區\nex：義大利麵 新北市新莊區\n\n3.傳送位置訊息")
			if err != nil {
				log.Println(err)
			}
		} else if content != nil && content.ContentType == linebot.ContentTypeLocation {
			//receive location
			// create a new yelp client with the auth keys
			client := yelp.New(o, nil)

			loc, err := content.LocationContent()
			if err != nil {
				log.Println(err)
			}

			// Build an advanced set of search criteria that include
			// general options, and coordinate options.
			s := yelp.SearchOptions{
				GeneralOptions: &yelp.GeneralOptions{
				    Term: "food,restaurants",
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
				_, err = bot.SendText([]string{content.From}, "查無資料！\n\n請輸入：\n1.地區\nex：台北市信義區\n\n2.食物 地區\nex：義大利麵 新北市新莊區\n\n3.傳送位置訊息")
			}

			for j := 0; j < 3; j++ {
				i := rand.Intn(20)
				urlOrig := UrlShortener{}
				urlOrig.short(results.Businesses[i].MobileURL)
				address := strings.Join(results.Businesses[i].Location.DisplayAddress, ",")
				var largeImageURL = strings.Replace(results.Businesses[i].ImageURL, "ms.jpg", "l.jpg", 1)

				_, err = bot.SendImage([]string{content.From}, largeImageURL, largeImageURL)
				_, err = bot.SendText([]string{content.From}, "店名："+results.Businesses[i].Name+"\n電話："+results.Businesses[i].Phone+"\n評比："+strconv.FormatFloat(float64(results.Businesses[i].Rating), 'f', 1, 64)+"\n更多資訊：" + urlOrig.ShortUrl)
				_, err = bot.SendLocation([]string{content.From}, results.Businesses[i].Name+"\n", address, float64(results.Businesses[i].Location.Coordinate.Latitude), float64(results.Businesses[i].Location.Coordinate.Longitude))
			}
		} else if content != nil && content.IsMessage && content.ContentType == linebot.ContentTypeText {
			//receive text
			// create a new yelp client with the auth keys
			client := yelp.New(o, nil)

			text, err := content.TextContent()
			c := strings.Split(text.Text, " ")

			if len(c) == 1 {
				// make a simple query for location
				results, err := client.DoSimpleSearch("food,restaurants", c[0])
				if err != nil {
					log.Println(err)
					_, err = bot.SendText([]string{content.From}, "查無資料！\n\n請輸入：\n1.地區\nex：台北市信義區\n\n2.食物 地區\nex：義大利麵 新北市新莊區\n\n3.傳送位置訊息")
				}

				for j := 0; j < 3; j++ {
					i := rand.Intn(20)
					urlOrig := UrlShortener{}
					urlOrig.short(results.Businesses[i].MobileURL)
					address := strings.Join(results.Businesses[i].Location.DisplayAddress, ",")
					var largeImageURL = strings.Replace(results.Businesses[i].ImageURL, "ms.jpg", "l.jpg", 1)

					_, err = bot.SendImage([]string{content.From}, largeImageURL, largeImageURL)
					_, err = bot.SendText([]string{content.From}, "店名："+results.Businesses[i].Name+"\n電話："+results.Businesses[i].Phone+"\n評比："+strconv.FormatFloat(float64(results.Businesses[i].Rating), 'f', 1, 64)+"\n更多資訊：" + urlOrig.ShortUrl)
					_, err = bot.SendLocation([]string{content.From}, results.Businesses[i].Name+"\n", address, float64(results.Businesses[i].Location.Coordinate.Latitude), float64(results.Businesses[i].Location.Coordinate.Longitude))
				}
			} else if len(c) == 2 {
				// make a simple query for food and location
				results, err := client.DoSimpleSearch(c[0], c[1])
				if err != nil {
					log.Println(err)
					_, err = bot.SendText([]string{content.From}, "查無資料！\n\n請輸入：\n1.地區\nex：台北市信義區\n\n2.食物 地區\nex：義大利麵 新北市新莊區\n\n3.傳送位置訊息")
				}

				for j := 0; j < 3; j++ {
					i := rand.Intn(20)
					urlOrig := UrlShortener{}
					urlOrig.short(results.Businesses[i].MobileURL)
					address := strings.Join(results.Businesses[i].Location.DisplayAddress, ",")
					var largeImageURL = strings.Replace(results.Businesses[i].ImageURL, "ms.jpg", "l.jpg", 1)

					_, err = bot.SendImage([]string{content.From}, largeImageURL, largeImageURL)
					_, err = bot.SendText([]string{content.From}, "店名："+results.Businesses[i].Name+"\n電話："+results.Businesses[i].Phone+"\n評比："+strconv.FormatFloat(float64(results.Businesses[i].Rating), 'f', 1, 64)+"\n更多資訊：" + urlOrig.ShortUrl)
					_, err = bot.SendLocation([]string{content.From}, results.Businesses[i].Name+"\n", address, float64(results.Businesses[i].Location.Coordinate.Latitude), float64(results.Businesses[i].Location.Coordinate.Longitude))
				}
			} else {
				_, err = bot.NewMultipleMessage().
					AddText("格式輸入錯誤！\n\n請輸入：\n1.地區\nex：台北市信義區\n\n2.食物 地區\nex：義大利麵 新北市新莊區\n\n3.傳送位置訊息").
					Send([]string{content.From})
			}
			if err != nil {
				log.Println(err)
			}
		}
	}
}