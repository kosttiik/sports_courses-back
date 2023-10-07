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
	Coach      string
	Phone      string
	Capacity   int
	Registered int
}

var cards = []Card{
	{"Футбол", "Стадион", "image/football.png", "Самый популярный вид спорта, в котором две команды соревнуются за то, чтобы забить мяч в ворота соперника, используя различные части тела, кроме рук и рукавиц", "Иванов Дмитрий Константинович", "+79167776968", 64, 54},
	{"Баскетбол", "Спорткомплекс", "image/basketball.png", "Игра с мячом, целью которой является забрасывание мяча в корзину соперника, расположенную на определенной высоте.", "Петров Виктор Петрович", "+79155679888", 32, 19},
	{"Теннис", "Манеж", "image/tennis.png", "Игра с ракетками, в которой два или четыре игрока ударяют мяч по корту, стремясь выиграть очки и сеты", "Чепушев Антон Викторович", "+79250102015", 32, 12},
	{"Тяжёлая атлетика", "Тренажёрный зал", "image/weightlifting.png", "Вид спорта, в котором спортсмены соревнуются в силовых упражнениях, таких как поднимание тяжестей (жим, толчок и рывок), демонстрируя максимальную физическую мощь", "Лаптев Григорий Иванович", "+79996530122", 20, 17},
	{"Плавание", "Бассейн", "image/swimming-pool.png", "Спортивная дисциплина, в которой участники преодолевают водную дистанцию, используя различные стили плавания (кроль, брасс, баттерфляй, спиной), соревнуясь на время", "Шипов Тимофей Александрович", "+79102001010", 14, 14},
	{"Дзюдо", "Манеж", "image/judo.png", "Вид спорта, в котором спортсмены соревнуются, пытаясь выиграть бой, бросив соперника на мат или зафиксировав его в положении на спине", "Аракелян Петр Михайлович", "+79254839154", 16, 7},
}

func StartServer() {
	log.Println("Server is starting up...")

	r := gin.Default()

	r.LoadHTMLGlob("templates/*.html")
	r.Static("/image", "resources/image")
	r.Static("/css", "templates/css")
	r.Static("/font", "resources/font")

	r.GET("/", loadCourses)
	r.GET("/:title", loadCourse)

	r.Run()

	log.Println("Server shutdown.")
}

func loadCourses(c *gin.Context) {
	course_title := c.Query("course_title")

	if course_title == "" {
		c.HTML(http.StatusOK, "courses.html", gin.H{
			"cards": cards,
		})
		return
	}

	foundCards := []Card{}
	lowerCardTitle := strings.ToLower(course_title)
	for i := range cards {
		if strings.Contains(strings.ToLower(cards[i].Title), lowerCardTitle) {
			foundCards = append(foundCards, cards[i])
		}
	}

	c.HTML(http.StatusOK, "courses.html", gin.H{
		"cards":  foundCards,
		"Search": course_title,
	})
}

func loadCourse(c *gin.Context) {
	title := c.Param("title")

	for i := range cards {
		if cards[i].Title == title {
			c.HTML(http.StatusOK, "course.html", gin.H{
				"Title":      cards[i].Title,
				"Text":       cards[i].Text,
				"Image":      "../" + cards[i].Image,
				"Details":    cards[i].Details,
				"Coach":      cards[i].Coach,
				"Phone":      cards[i].Phone,
				"Capacity":   cards[i].Capacity,
				"Registered": cards[i].Registered,
			})
			return
		}
	}
}
