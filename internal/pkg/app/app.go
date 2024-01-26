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
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

	// swagger
	docs.SwaggerInfo.BasePath = "/"
	a.r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	a.r.Use(a.WithAuthCheck(role.Moderator, role.Admin, role.User, role.Undefined)).GET("groups", a.get_groups)
	a.r.GET("group/:group", a.get_group)

	// authorization
	a.r.POST("/login", a.login)
	a.r.POST("/register", a.register)
	a.r.POST("/logout", a.logout)

	a.r.Use(a.WithAuthCheck(role.Moderator, role.Admin, role.User)).GET("enrollment", a.get_enrollment)
	a.r.POST("group/add_to_enrollment/:id", a.add_group_to_enrollment)
	a.r.DELETE("enrollment_to_group/delete", a.delete_enrollment_to_group)
	a.r.GET("enrollments", a.get_enrollments)
	a.r.PUT("enrollment/edit", a.edit_enrollment)
	a.r.PUT("enroll", a.enroll)
	a.r.PUT("enrollment/status_change", a.enrollment_status_change)
	a.r.DELETE("enrollment/delete/:enrollment_id", a.delete_enrollment)
	a.r.PUT("enrollment/user_confirm/:enrollment_id", a.user_confirm_enrollment)
	a.r.PUT("enrollment_to_group/set_group_availability", a.enrollment_to_group_set_group_availability)
	a.r.GET("enrollment_groups/:enrollment_id", a.enrollment_groups)
	a.r.PUT("enrollment/set_groups", a.set_enrollment_groups)

	a.r.Use(a.WithAuthCheck(role.Moderator, role.Admin)).POST("group/add_image/:group_id", a.add_image)
	a.r.PUT("enrollment/moderator_confirm/:enrollment_id", a.moderator_confirm_enrollment)
	a.r.DELETE("group/delete/:group_title", a.delete_group)
	a.r.PUT("group/edit", a.edit_group)
	a.r.POST("group/add", a.add_group)

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
	var status = c.Query("status")

	groups, err := a.repo.GetGroups(title_pattern, course, status)
	if err != nil {
		c.Error(err)
		return
	}

	_userUUID, ok := c.Get("userUUID")

	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"groups": groups,
		})
		return
	}

	userUUID := _userUUID.(uuid.UUID)

	draft_enrollment, err := a.repo.GetDraftEnrollment(userUUID)

	if err != nil {
		log.Println(err)
		c.String(http.StatusInternalServerError, "Возникла ошибка при поиске заявки-черновика")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groups":           groups,
		"draft_enrollment": draft_enrollment,
	})
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
	startDate := c.Query("startDate")
	endDate := c.Query("endDate")

	enrollments, err := a.repo.GetEnrollments(status, startDate, endDate, roleNumber, userUUID)
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

	c.JSON(http.StatusOK, found_enrollment)
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

	// _userUUID, _ := c.Get("userUUID")
	// userUUID := _userUUID.(uuid.UUID)

	var enrollment = ds.Enrollment{}
	enrollment.ID = uint(requestBody.EnrollmentID)

	err := a.repo.EditEnrollment(&enrollment)

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

	status, err := a.repo.GetEnrollmentStatus(requestBody.EnrollmentID)
	if err == nil {
		c.Error(err)
		return
	}

	if userRole == role.User && requestBody.Status == "Удалён" {
		if status == "Черновик" || status == "Сформирован" {
			err = a.repo.ChangeEnrollmentStatusUser(requestBody.EnrollmentID, requestBody.Status, userUUID)

			if err != nil {
				c.Error(err)
				return
			} else {
				c.String(http.StatusCreated, "Статус записи был успешно обновлён")
			}
		}
	} else {
		err = a.repo.ChangeEnrollmentStatus(requestBody.EnrollmentID, requestBody.Status)

		if err != nil {
			c.Error(err)
			return
		}

		if userRole == role.Moderator && status == "Черновик" {
			err = a.repo.SetEnrollmentModerator(requestBody.EnrollmentID, userUUID)

			if err != nil {
				c.Error(err)
				return
			}
		}

		c.String(http.StatusCreated, "Статус записи был успешно обновлён")
	}
}

type changeEnrollmentToGroupAvailabilityReq struct {
	enrollmentToGroupId int
	availability        string
}

// @Summary      Редактировать статус м-м
// @Description  Получает id записи м-м и новый статус и производит необходимые обновления
// @Tags         Запись
// @Accept json
// @Produce json
// @Success 201 {object} string
// @Param request_body body ds.ChangeEnrollmentToGroupAvailabilityRequestBody true "Request body"
// @Router /enrollment/set_group_availability [put]
func (a *Application) enrollment_to_group_set_group_availability(c *gin.Context) {
	req := &changeEnrollmentToGroupAvailabilityReq{}
	err := json.NewDecoder(c.Request.Body).Decode(req)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	enrollment_to_group := &ds.EnrollmentToGroup{}
	enrollment_to_group.ID = uint(req.enrollmentToGroupId)
	enrollment_to_group.Availability = req.availability

	// _userUUID, _ := c.Get("userUUID")
	// userUUID := _userUUID.(uuid.UUID)

	err = a.repo.ChangeEnrollmentToGroupAvailability(enrollment_to_group)
	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusOK, "М-М был успешно обновлен")
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
	group_param := c.Query("group_id")
	enrollment_param := c.Query("enrollment_id")

	group_id, err := strconv.Atoi(group_param)
	enrollment_id, err := strconv.Atoi(enrollment_param)

	if err != nil {
		c.String(http.StatusBadRequest, "Переданы некорректные ID")
		return
	}

	err = a.repo.DeleteEnrollmentToGroup(enrollment_id, group_id)

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
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("пароль не может быть пустым"))
		return
	}
	if req.Login == "" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("имя не может быть пустым"))
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

func (a *Application) moderator_confirm_enrollment(c *gin.Context) {
	id_param := c.Param("enrollment_id")
	enrollment_id, err := strconv.Atoi(id_param)
	if err != nil {
		c.String(http.StatusBadRequest, "Передан некорректный ID записи")
		return
	}

	confirm_param := c.Query("confirm")
	confirm := true
	if confirm_param == "True" {
		confirm = true
	} else if confirm_param == "False" {
		confirm = false
	} else {
		c.String(http.StatusBadRequest, "Передан некорректный флаг подтверждения")
		return
	}

	_userUUID, _ := c.Get("userUUID")
	userUUID := _userUUID.(uuid.UUID)

	err = a.repo.ModeratorConfirmEnrollment(userUUID, enrollment_id, confirm)
	if err != nil {
		c.String(http.StatusInternalServerError, "Не получается обновить статус!")
		return
	}

	c.String(http.StatusOK, "Статус обновлён!")
}

func (a *Application) user_confirm_enrollment(c *gin.Context) {
	id_param := c.Param("enrollment_id")
	enrollment_id, err := strconv.Atoi(id_param)

	if err != nil {
		c.String(http.StatusBadRequest, "Передан некорректный ID записи")
		return
	}

	_userUUID, _ := c.Get("userUUID")
	userUUID := _userUUID.(uuid.UUID)

	err = a.repo.UserConfirmEnrollment(userUUID, enrollment_id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Не получается обновить статус!")
		return
	}

	c.String(http.StatusOK, "Статус обновлён!")
}

func (a *Application) add_group_to_enrollment(c *gin.Context) {
	group_param := c.Param("id")

	group_id, err := strconv.Atoi(group_param)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	_userUUID, ok := c.Get("userUUID")
	if !ok {
		c.String(http.StatusInternalServerError, "Не могу распознать uuid")
		log.Println(_userUUID)
		return
	}
	userUUID := _userUUID.(uuid.UUID)

	draft, err := a.repo.GetDraftEnrollment(userUUID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Не могу найти черновую запись!")
	}

	if draft.Status == "" {
		new_draft := ds.Enrollment{}
		new_draft.UserRefer = &userUUID
		new_draft.DateCreated = time.Now()
		new_draft.Status = "Черновик"
		new_draft.ModeratorRefer = nil

		err := a.repo.CreateEnrollment(new_draft)
		if err != nil {
			c.String(http.StatusInternalServerError, "Не могу создать черновую запись!")
			return
		}

		draft, err = a.repo.GetDraftEnrollment(userUUID)
		if err != nil {
			c.String(http.StatusInternalServerError, "Не могу найти черновую запись!")
		}
	}

	group_to_draft := ds.EnrollmentToGroup{}
	group_to_draft.EnrollmentRefer = int(draft.ID)
	group_to_draft.GroupRefer = group_id

	err = a.repo.CreateEnrollmentToGroup(group_to_draft)
	if err != nil {
		c.String(http.StatusInternalServerError, "Не могу связать группу с записью!")
	}

	c.String(http.StatusOK, "Группа добавлена в черновую запись!")
}

func (a *Application) add_image(c *gin.Context) {
	group_id, err := strconv.Atoi(c.Param("group_id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Не получается прочитать ID группы")
		log.Println("Не получается прочитать ID группы")
		return
	}

	image, header, err := c.Request.FormFile("file")

	if err != nil {
		c.String(http.StatusBadRequest, "Не получается распознать картинку")
		log.Println("Не получается распознать картинку")
		return
	}
	defer image.Close()

	minioClient, err := minio.New("127.0.0.1:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})

	if err != nil {
		c.String(http.StatusInternalServerError, "Не получается подключиться к minio")
		log.Println("Не получается подключиться к minio")
		return
	}

	objectName := header.Filename
	_, err = minioClient.PutObject(c.Request.Context(), "groupimages", objectName, image, header.Size, minio.PutObjectOptions{})

	if err != nil {
		c.String(http.StatusInternalServerError, "Не получилось загрузить картинку в minio")
		log.Println("Не получилось загрузить картинку в minio")
		return
	}

	err = a.repo.SetGroupImage(group_id, objectName)

	if err != nil {
		c.String(http.StatusInternalServerError, "Не получается обновить картинку группы")
		log.Println("Не получается обновить картинку группы")
		return
	}

	c.String(http.StatusCreated, "Картинка загружена!")
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
