package api

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Card struct {
	Title      string
	Text       string
	Image      string
	Details    string
	Capacity   int
	Registered int
}

var cards = []Card{
	{"Футбол", "Стадион", "image/image.jpg", "Самый популярный вид спорта", 32, 15},
	{"Баскетбол", "Спорткомплекс", "image/image.jpg", "Самый популярный вид спорта", 32, 19},
	{"Теннис", "Манеж", "image/image.jpg", "Самый популярный вид спорта", 32, 12},
	{"Спортивное ориентирование", "Измайлово", "image/image.jpg", "Самый популярный вид спорта", 32, 17},
	{"Плавание", "Бассейн", "image/image.jpg", "Самый популярный вид спорта", 32, 14},
}

func StartServer() {
	log.Println("Server is starting up...")

	r := gin.Default()

	r.LoadHTMLGlob("templates/*.html")
	r.Static("/image", "resources")
	r.Static("/css", "templates/css")
	r.Static("/font", "resources/font")

	r.GET("/", loadHome)
	r.GET("/:title", loadPage)

	r.Run()

	log.Println("Server shutdown.")
}

func loadHome(c *gin.Context) {
	card_title := c.Query("card_title")

	if card_title == "" {
		c.HTML(http.StatusOK, "courses.html", gin.H{
			"cards": cards,
		})
		return
	}

	foundCards := []Card{}
	lowerCardTitle := strings.ToLower(card_title)
	for i := range cards {
		if strings.Contains(strings.ToLower(cards[i].Title), lowerCardTitle) {
			foundCards = append(foundCards, cards[i])
		}
	}

	c.HTML(http.StatusOK, "courses.html", gin.H{
		"cards": foundCards,
	})
}

func loadPage(c *gin.Context) {
	title := c.Param("title")

	for i := range cards {
		if cards[i].Title == title {
			c.HTML(http.StatusOK, "region.html", gin.H{
				"Title":      cards[i].Title,
				"Text":       cards[i].Text,
				"Image":      "../" + cards[i].Image,
				"Details":    cards[i].Details,
				"Capacity":   cards[i].Capacity,
				"Registered": cards[i].Registered,
			})
			return
		}
	}
}
