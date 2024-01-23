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

	a.r.GET("groups", a.get_groups)
	a.r.GET("group/:group", a.get_group)

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
	a.r.PUT("enrollment_to_group/status_change", a.enrollment_to_group_status_change)
	a.r.GET("enrollment_groups/:enrollment_id", a.enrollment_groups)
	a.r.PUT("enrollment/set_groups", a.set_enrollment_groups)

	a.r.Use(a.WithAuthCheck(role.Moderator, role.Admin)).PUT("group/delete_restore/:group_title", a.delete_restore_group)
	a.r.PUT("enrollment/delete/:enrollment_id", a.delete_enrollment)
	a.r.PUT("enrollment_to_group/delete", a.delete_enrollment_to_group)
	a.r.PUT("enrollment/edit", a.edit_enrollment)
	a.r.PUT("group/delete/:group_title", a.delete_group)
	a.r.PUT("group/edit", a.edit_group)
	a.r.PUT("group/add", a.add_group)

	a.r.Run()

	log.Println("Server shutdown.")
}

// @Summary Получить все существующие группы
// @Description Возвращает все существующие группы
// @Tags Группы
// @Accept json
// @Produce json
// @Success 200 {} json
// @Param name_pattern query string false "Паттерн названия группы"
// @Param location query string false "Локация"
// @Param status query string false "Статус группы (Действует/Недействителен)"
// @Router /groups [get]
func (a *Application) get_groups(c *gin.Context) {
	var title_pattern = c.Query("title_pattern")
	var course = c.Query("course")
	var location = c.Query("location")
	var status = c.Query("status")

	groups, err := a.repo.GetAllGroups(title_pattern, course, location, status)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, groups)
}

// @Summary      Добавляет новую группу в БД
// @Description  Создает новую группу с параметрами, описанными в json'е
// @Tags Группы
// @Accept json
// @Produce      json
// @Param group body ds.Group true "Характеристики новой группы"
// @Success      201  {object}  string "Группа успешно добавлена"
// @Router       /group/add [put]
func (a *Application) add_group(c *gin.Context) {
	var group ds.Group

	if err := c.BindJSON(&group); err != nil || group.Title == "" || group.Status == "" {
		c.String(http.StatusBadRequest, "Не получается распознать группу\n"+err.Error())
		return
	}

	if group.Status == "" {
		group.Status = "Черновик"
	}

	err := a.repo.CreateGroup(group)

	if err != nil {
		c.String(http.StatusNotFound, "Не получается создать группу\n"+err.Error())
		return
	}

	c.String(http.StatusCreated, "Группа успешно добавлена")

}

// @Summary      Получить группу
// @Description  Возвращает данные группы с переданным названием
// @Tags         Группы
// @Produce      json
// @Success      200  {object}  string
// @Router       /group/{group} [get]
func (a *Application) get_group(c *gin.Context) {
	var group = ds.Group{}

	group.Title = c.Param("group")

	found_group, err := a.repo.FindGroup(group)

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, found_group)

}

// @Summary      Редактировать группу
// @Description  Находит группу по имени и обновляет перечисленные поля
// @Tags         Группы
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Param group body ds.Group true "Данные редактируемого группы (должны содержать имя группы или его id)"
// @Router       /group/edit [put]
func (a *Application) edit_group(c *gin.Context) {
	var group *ds.Group

	if err := c.BindJSON(&group); err != nil {
		c.Error(err)
		return
	}

	err := a.repo.EditGroup(group)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Группа была успешно изменена")
}

// @Summary      Удалить группу
// @Description  Находит группу по его названию и изменяет его статус на "Недоступен"
// @Tags         Группы
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Param group_title path string true "Название группы"
// @Router       /group/delete/{group_title} [put]
func (a *Application) delete_group(c *gin.Context) {
	group_title := c.Param("group_title")

	if group_title == "" {
		c.String(http.StatusBadRequest, "Вы должны указать паттерн названия группы")

		return
	}

	err := a.repo.LogicalDeleteGroup(group_title)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusFound, "Группа был успешно удалена")
}

// @Summary      Удалить или восстановить группу
// @Description  Изменяет статус группы с "Действует" на "Недоступен" и обратно
// @Tags         Группы
// @Produce      json
// @Success      200  {object}  string
// @Param group_title path string true "Название группы"
// @Router       /group/delete_restore/{group_title} [get]
func (a *Application) delete_restore_group(c *gin.Context) {
	group_title := c.Param("group_title")

	if group_title == "" {
		c.String(http.StatusBadRequest, "Вы должны указать паттерн названия группы")
	}

	err := a.repo.DeleteRestoreGroup(group_title)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusFound, "Статус группы был успешно изменён")
}

// @Summary      Записать в группу/ы
// @Description  Создаёт новую заявку и связывает её с группой/ами
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
		c.String(http.StatusNotFound, "Не могу записаться в группу")
		return
	}

	c.String(http.StatusCreated, "Запись в группу прошла успешно")
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
	enrollment.ID = uint(requestBody.EnrollmentID)
	enrollment.Status = requestBody.Status

	err := a.repo.EditEnrollment(&enrollment, userUUID)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Запись была успешна обновлена")
}

func (a *Application) enrollment_groups(c *gin.Context) {
	enrollment_id, err := strconv.Atoi(c.Param("enrollment_id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Не могу разобрать id записи!")
		return
	}

	groups, err := a.repo.GetEnrollmentGroups(enrollment_id)
	log.Println(groups)
	if err != nil {
		c.String(http.StatusInternalServerError, "Не получается узнать группы связанные с записью!")
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (a *Application) set_enrollment_groups(c *gin.Context) {
	var requestBody ds.SetEnrollmentGroupsRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.String(http.StatusBadRequest, "Не получается распознать json запрос")
		return
	}

	err := a.repo.SetEnrollmentGroups(requestBody.EnrollmentID, requestBody.Groups)
	if err != nil {
		c.String(http.StatusInternalServerError, "Не получилось задать группы для записи\n"+err.Error())
	}

	c.String(http.StatusCreated, "Группы записи успешно заданы!")
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

// @Summary      Редактировать статус м-м
// @Description  Получает id записи м-м и новый статус и производит необходимые обновления
// @Tags         Запись
// @Accept json
// @Produce json
// @Success 201 {object} string
// @Param request_body body ds.ChangeEnrollmentToGroupStatusRequestBody true "Request body"
// @Router /enrollment/status_change [put]
func (a *Application) enrollment_to_group_status_change(c *gin.Context) {
	var requestBody ds.ChangeEnrollmentToGroupStatusRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.String(http.StatusBadRequest, "Передан плохой json")
		return
	}

	var enrollment_to_group = ds.EnrollmentToGroup{}
	enrollment_to_group.ID = uint(requestBody.ID)
	enrollment_to_group.Status = requestBody.Status

	err := a.repo.ChangeEnrollmentToGroupStatus(int(enrollment_to_group.ID), enrollment_to_group.Status)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Заявка была успешно обновлена")
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

// @Summary      Удаляет связь группы с записью
// @Description  Удаляет запись в таблице enrollment_to_group
// @Tags         enrollments
// @Accept json
// @Produce      json
// @Success      201  {object}  string
// @Param request_body body ds.DeleteEnrollmentToGroupRequestBody true "Параметры запроса"
// @Router       /enrollment_to_group/delete [put]
func (a *Application) delete_enrollment_to_group(c *gin.Context) {
	var requestBody ds.DeleteEnrollmentToGroupRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.Error(err)
		return
	}

	err := a.repo.DeleteEnrollmentToGroup(requestBody.EnrollmentID, requestBody.GroupID)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Связь группы с записью была успешно удалена")
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
