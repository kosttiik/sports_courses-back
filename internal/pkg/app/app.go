package app

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sports_courses/docs"
	"sports_courses/internal/app/config"
	"sports_courses/internal/app/ds"
	"sports_courses/internal/app/dsn"
	"sports_courses/internal/app/redis"
	"sports_courses/internal/app/repository"
	"sports_courses/internal/app/role"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/datatypes"

	"github.com/gin-gonic/gin"
)

// @BasePath /

type Application struct {
	repo   *repository.Repository
	r      *gin.Engine
	config *config.Config
	redis  *redis.Client
}

type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResp struct {
	Login       string `json:"login"`
	Role        int    `json:"role"`
	ExpiresIn   int    `json:"expires_in"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func New(ctx context.Context) (*Application, error) {
	cfg, err := config.NewConfig(ctx)
	if err != nil {
		return nil, err
	}

	repo, err := repository.New(dsn.FromEnv())
	if err != nil {
		return nil, err
	}

	redisClient, err := redis.New(ctx, cfg.Redis)
	if err != nil {
		return nil, err
	}

	return &Application{
		config: cfg,
		repo:   repo,
		redis:  redisClient,
	}, nil
}

func (a *Application) StartServer() {
	log.Println("Server is starting up...")

	a.r = gin.Default()

	a.r.GET("courses", a.get_courses)
	a.r.GET("course/:course", a.get_course)

	// swagger
	docs.SwaggerInfo.BasePath = "/"
	a.r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// registration and login
	a.r.POST("/login", a.login)
	a.r.POST("/register", a.register)
	a.r.POST("/logout", a.logout)

	a.r.Use(a.WithAuthCheck(role.Moderator, role.Admin, role.User)).GET("enrollment", a.get_enrollment)
	a.r.GET("enrollments", a.get_enrollments)
	a.r.PUT("enroll", a.enroll)
	a.r.PUT("enrollment/status_change", a.enrollment_status_change)
	a.r.GET("enrollment_courses/:enrollment_id", a.enrollment_courses)
	a.r.PUT("enrollment/set_courses", a.set_enrollment_courses)

	a.r.Use(a.WithAuthCheck(role.Moderator, role.Admin)).PUT("course/delete_restore/:course_title", a.delete_restore_course)
	a.r.PUT("enrollment/delete/:enrollment_id", a.delete_enrollment)
	a.r.PUT("enrollment_to_course/delete", a.delete_enrollment_to_course)
	a.r.PUT("enrollment/edit", a.edit_enrollment)
	a.r.PUT("course/delete/:course_title", a.delete_course)
	a.r.PUT("course/edit", a.edit_course)
	a.r.PUT("course/add", a.add_course)

	a.r.Run()

	log.Println("Server shutdown.")
}

// @Summary Получить все существующие курсы
// @Description Возвращает все существующие курсы
// @Tags Курсы
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param name_pattern query string false "Паттерн названия курса"
// @Param location query string false "Локация"
// @Param status query string false "Статус курса (Действует/Недействителен)"
// @Router /courses [get]
func (a *Application) get_courses(c *gin.Context) {
	var title_pattern = c.Query("title_pattern")
	var location = c.Query("location")
	var status = c.Query("status")

	courses, err := a.repo.GetAllCourses(title_pattern, location, status)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, courses)
}

// @Summary      Добавляет новый курс в БД
// @Description  Создает новый курс с параметрами, описанными в json'е
// @Tags Курсы
// @Accept json
// @Produce      json
// @Param course body ds.Course true "Характеристики нового курса"
// @Success      201  {object}  string "Курс успешно добавлен"
// @Router       /course/add [put]
func (a *Application) add_course(c *gin.Context) {
	var course ds.Course

	if err := c.BindJSON(&course); err != nil || course.Title == "" || course.Status == "" {
		c.String(http.StatusBadRequest, "Не получается распознать курс\n"+err.Error())
		return
	}

	if course.Status == "" {
		course.Status = "Черновик"
	}

	err := a.repo.CreateCourse(course)

	if err != nil {
		c.String(http.StatusNotFound, "Не получается создать курс\n"+err.Error())
		return
	}

	c.String(http.StatusCreated, "Курс успешно добавлен")

}

// @Summary      Получить курс
// @Description  Возвращает данные курса с переданным названием
// @Tags         Курсы
// @Produce      json
// @Success      200  {object}  string
// @Router       /course/{course} [get]
func (a *Application) get_course(c *gin.Context) {
	var course = ds.Course{}

	course.Title = c.Param("course")

	found_course, err := a.repo.FindCourse(course)

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, found_course)

}

// @Summary      Редактировать курс
// @Description  Находит курс по имени и обновляет перечисленные поля
// @Tags         Курсы
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Param course body ds.Course true "Данные редактируемого курса (должны содержать имя курса или его id)"
// @Router       /course/edit [put]
func (a *Application) edit_course(c *gin.Context) {
	var course *ds.Course

	if err := c.BindJSON(&course); err != nil {
		c.Error(err)
		return
	}

	err := a.repo.EditCourse(course)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Курс был успешно изменён")

}

// @Summary      Удалить курс
// @Description  Находит курс по его названию и изменяет его статус на "Недоступен"
// @Tags         Курсы
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Param course_title path string true "Название курса"
// @Router       /course/delete/{course_title} [put]
func (a *Application) delete_course(c *gin.Context) {
	course_title := c.Param("course_title")

	if course_title == "" {
		c.String(http.StatusBadRequest, "Вы должны указать паттерн названия курса")

		return
	}

	err := a.repo.LogicalDeleteCourse(course_title)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusFound, "Курс был успешно удалён")
}

// @Summary      Удалить или восстановить курс
// @Description  Изменяет статус курса с "Действует" на "Недоступен" и обратно
// @Tags         Курсы
// @Produce      json
// @Success      200  {object}  string
// @Param course_title path string true "Название курса"
// @Router       /course/delete_restore/{course_title} [get]
func (a *Application) delete_restore_course(c *gin.Context) {
	course_title := c.Param("course_title")

	if course_title == "" {
		c.String(http.StatusBadRequest, "Вы должны указать паттерн названия курса")
	}

	err := a.repo.DeleteRestoreCourse(course_title)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusFound, "Статус курса был успешно изменён")
}

// @Summary      Записать на курс/сы
// @Description  Создаёт новую заявку и связывает её с курсом/ами
// @Tags Запись
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Param Body body ds.EnrollRequestBody true "Параметры записи"
// @Router       /enroll [put]
func (a *Application) enroll(c *gin.Context) {
	var request_body ds.EnrollRequestBody

	if err := c.BindJSON(&request_body); err != nil {
		c.String(http.StatusBadGateway, "Не могу распознать json")
		return
	}

	_userUUID, ok := c.Get("userUUID")

	if !ok {
		c.String(http.StatusInternalServerError, "Сначала Вам нужно авторизоваться")
		return
	}

	userUUID := _userUUID.(uuid.UUID)
	err := a.repo.Enroll(request_body, userUUID)

	if err != nil {
		c.Error(err)
		c.String(http.StatusNotFound, "Не могу записаться на курс")
		return
	}

	c.String(http.StatusCreated, "Запись на курс прошла успешно")
}

// @Summary      Получить записи
// @Description  Возвращает список всех доступных записей
// @Tags         Записи
// @Produce      json
// @Success      302  {object}  string
// @Param status query string false "Статус записи"
// @Router       /enrollments [get]
func (a *Application) get_enrollments(c *gin.Context) {
	_roleNumber, _ := c.Get("role")
	_userUUID, _ := c.Get("userUUID")

	roleNumber := _roleNumber.(role.Role)
	userUUID := _userUUID.(uuid.UUID)

	status := c.Query("status")

	enrollments, err := a.repo.GetAllEnrollments(status, roleNumber, userUUID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, enrollments)
}

// @Summary      Получить запись
// @Description  Возвращает запись с переданными параметрами
// @Tags         Записи
// @Accept		 json
// @Produce      json
// @Success      302  {object}  string
// @Param id query id false "ID записи"
// @Router       /enrollment [get]
func (a *Application) get_enrollment(c *gin.Context) {
	status := c.Query("status")
	id, _ := strconv.ParseUint(c.Query("enrollment_id"), 10, 64)

	enrollment := &ds.Enrollment{
		Status: status,
		ID:     uint(id),
	}

	found_enrollment, err := a.repo.FindEnrollment(enrollment)

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusFound, found_enrollment)
}

// @Summary      Редактировать запись
// @Description  Находит запись и редактирует её поля
// @Tags         Записи
// @Accept json
// @Produce      json
// @Success      201  {object}  string
// @Param enrollment body ds.Enrollment false "Запись"
// @Router       /enrollment/edit [put]
func (a *Application) edit_enrollment(c *gin.Context) {
	var requestBody ds.EditEnrollmentRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.String(http.StatusBadRequest, "Передан плохой json")
		return
	}

	_userUUID, _ := c.Get("userUUID")
	userUUID := _userUUID.(uuid.UUID)

	var enrollment = ds.Enrollment{}
	enrollment.StartDate = datatypes.Date(requestBody.StartDate)
	enrollment.EndDate = datatypes.Date(requestBody.EndDate)
	enrollment.ID = uint(requestBody.EnrollmentID)
	enrollment.Status = requestBody.Status

	err := a.repo.EditEnrollment(&enrollment, userUUID)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Запись была успешна обновлена")
}

func (a *Application) enrollment_courses(c *gin.Context) {
	enrollment_id, err := strconv.Atoi(c.Param("enrollment_id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Не могу разобрать id записи!")
		return
	}

	courses, err := a.repo.GetEnrollmentCourses(enrollment_id)
	log.Println(courses)
	if err != nil {
		c.String(http.StatusInternalServerError, "Не получается узнать курсы связанные с записью!")
		return
	}
	c.JSON(http.StatusOK, courses)
}

func (a *Application) set_enrollment_courses(c *gin.Context) {
	var requestBody ds.SetEnrollmentCoursesRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.String(http.StatusBadRequest, "Не получается распознать json запрос")
		return
	}

	err := a.repo.SetEnrollmentCourses(requestBody.EnrollmentID, requestBody.Courses)
	if err != nil {
		c.String(http.StatusInternalServerError, "Не получилось задать курсы для записи\n"+err.Error())
	}

	c.String(http.StatusCreated, "Курсы записи успешно заданы!")
}

// @Summary      Редактировать статус записи
// @Description  Получает id заявки и новый статус и производит необходимые обновления
// @Tags         Запись
// @Accept json
// @Produce json
// @Success 201 {object} string
// @Param request_body body ds.ChangeEnrollmentStatusRequestBody true "Request body"
// @Router /enrollment/status_change [put]
func (a *Application) enrollment_status_change(c *gin.Context) {
	var requestBody ds.ChangeEnrollmentStatusRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.Error(err)
		return
	}

	_userUUID, _ := c.Get("userUUID")
	_userRole, _ := c.Get("role")

	userUUID := _userUUID.(uuid.UUID)
	userRole := _userRole.(role.Role)

	status, err := a.repo.GetEnrollmentStatus(requestBody.ID)
	if err == nil {
		c.Error(err)
		return
	}

	if userRole == role.User && requestBody.Status == "Удалён" {
		if status == "Черновик" || status == "Сформирован" {
			err = a.repo.ChangeEnrollmentStatusUser(requestBody.ID, requestBody.Status, userUUID)

			if err != nil {
				c.Error(err)
				return
			} else {
				c.String(http.StatusCreated, "Статус записи был успешно обновлён")
			}
		}
	} else {
		err = a.repo.ChangeEnrollmentStatus(requestBody.ID, requestBody.Status)

		if err != nil {
			c.Error(err)
			return
		}

		if userRole == role.Moderator && status == "Черновик" {
			err = a.repo.SetEnrollmentModerator(requestBody.ID, userUUID)

			if err != nil {
				c.Error(err)
				return
			}
		}

		c.String(http.StatusCreated, "Статус записи был успешно обновлён")
	}
}

// @Summary      Удалить запись
// @Description  Изменяет статус записи на "Удалён"
// @Tags         Записи
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Param enrollment_id path int true "id записи"
// @Router       /enrollment/delete/{enrollment_id} [put]
func (a *Application) delete_enrollment(c *gin.Context) {
	enrollment_id, _ := strconv.Atoi(c.Param("enrollment_id"))

	err := a.repo.LogicalDeleteEnrollment(enrollment_id)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusFound, "Запись была успешно удалена")
}

// @Summary      Удаляет связь курса с записью
// @Description  Удаляет запись в таблице enrollment_to_course
// @Tags         enrollments
// @Accept json
// @Produce      json
// @Success      201  {object}  string
// @Param request_body body ds.DeleteEnrollmentToCourseRequestBody true "Параметры запроса"
// @Router       /enrollment_to_course/delete [put]
func (a *Application) delete_enrollment_to_course(c *gin.Context) {
	var requestBody ds.DeleteEnrollmentToCourseRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.Error(err)
		return
	}

	err := a.repo.DeleteEnrollmentToCourse(requestBody.EnrollmentID, requestBody.CourseID)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Связь курса с записью была успешно удалена")
}

// @Summary Вход в систему
// @Description Проверяет данные для входа и в случае успеха возвращает токен для входа
// @Tags Аутентификация
// @Produce json
// @Accept json
// @Success 200 {object} loginResp
// @Param request_body body loginReq true "Данные для входа"
// @Router /login [post]
func (a *Application) login(c *gin.Context) {
	req := &loginReq{}

	err := json.NewDecoder(c.Request.Body).Decode(req)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	user, err := a.repo.GetUserByLogin(req.Login)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if req.Login == user.Name && user.Pass == generateHashString(req.Password) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, &ds.JWTClaims{
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(3600000000000).Unix(),
				IssuedAt:  time.Now().Unix(),
				Issuer:    "kostik",
			},
			UserUUID: user.UUID,
			Scopes:   []string{}, // test data
			Role:     user.Role,
		})

		if token == nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("токен равен nil"))

			return
		}

		jwtToken := "test"

		strToken, err := token.SignedString([]byte(jwtToken))
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("не получается прочесть строку токена"))

			return
		}

		c.SetCookie("sports_courses-api-token", "Bearer "+strToken, 3600000000000, "", "", true, true)

		c.JSON(http.StatusOK, loginResp{
			Login:       user.Name,
			Role:        int(user.Role),
			ExpiresIn:   3600000000000,
			AccessToken: strToken,
			TokenType:   "Bearer",
		})

		return
	}

	c.AbortWithStatus(http.StatusForbidden)
}

type registerReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type registerResp struct {
	Ok bool `json:"ok"`
}

// @Summary Зарегистрировать нового пользователя
// @Description Добавляет в БД нового пользователя
// @Tags Аутентификация
// @Produce json
// @Accept json
// @Success 200 {object} registerResp
// @Param request_body body registerReq true "Данные для регистрации"
// @Router /register [post]
func (a *Application) register(c *gin.Context) {
	req := &registerReq{}
	err := json.NewDecoder(c.Request.Body).Decode(req)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if req.Password == "" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("password should not be empty"))
		return
	}
	if req.Login == "" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("name should not be empty"))
	}

	err = a.repo.Register(&ds.User{
		UUID: uuid.New(),
		Role: role.User,
		Name: req.Login,
		Pass: generateHashString(req.Password),
	})

	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, &registerResp{
		Ok: true,
	})
}

// @Summary Выйти из системы
// @Details Деактивирует текущий токен пользователя, добавляя его в блэклист в редисе
// @Tags Аутентификация
// @Produce json
// @Accept json
// @Success 200
// @Router /logout [post]
func (a *Application) logout(c *gin.Context) {
	jwtStr := c.GetHeader("Authorization")
	if !strings.HasPrefix(jwtStr, jwtPrefix) {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	jwtStr = jwtStr[len(jwtPrefix):]

	_, err := jwt.ParseWithClaims(jwtStr, &ds.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("test"), nil
	})
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		log.Println(err)

		return
	}

	err = a.redis.WriteJWTToBlackList(c.Request.Context(), jwtStr, 3600000000000)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	c.Status(http.StatusOK)
}

func generateHashString(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func createSignedTokenString() (string, error) {
	privateKey, err := ioutil.ReadFile("demo.rsa")
	if err != nil {
		return "", fmt.Errorf("error reading private key file: %v\n", err)
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		return "", fmt.Errorf("error parsing RSA private key: %v\n", err)
	}

	token := jwt.New(jwt.SigningMethodRS256)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("error signing token: %v\n", err)
	}

	return tokenString, nil
}
