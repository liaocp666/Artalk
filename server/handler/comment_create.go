package handler

import (
	"errors"
	"strings"

	"github.com/ArtalkJS/Artalk/internal/core"
	"github.com/ArtalkJS/Artalk/internal/entity"
	"github.com/ArtalkJS/Artalk/internal/i18n"
	"github.com/ArtalkJS/Artalk/internal/log"
	"github.com/ArtalkJS/Artalk/internal/utils"
	"github.com/ArtalkJS/Artalk/server/common"
	"github.com/gofiber/fiber/v2"
)

type ParamsCommentCreate struct {
	Name    string `json:"name" validate:"required"`    // The comment name
	Email   string `json:"email" validate:"required"`   // The comment email
	Link    string `json:"link" validate:"optional"`    // The comment link
	Content string `json:"content" validate:"required"` // The comment content
	Rid     uint   `json:"rid" validate:"optional"`     // The comment rid
	UA      string `json:"ua" validate:"optional"`      // The comment ua

	PageKey   string `json:"page_key" validate:"required"`   // The comment page_key
	PageTitle string `json:"page_title" validate:"optional"` // The comment page_title

	SiteName string `json:"site_name" validate:"required"` // The site name of your content scope
}

type ResponseCommentCreate struct {
	entity.CookedComment
}

// @Id           CreateComment
// @Summary      Create Comment
// @Description  Create a new comment
// @Tags         Comment
// @Param        comment  body  ParamsCommentCreate  true  "The comment data"
// @Security     ApiKeyAuth
// @Success      200  {object}  ResponseCommentCreate
// @Failure      400  {object}  Map{msg=string}
// @Failure      500  {object}  Map{msg=string}
// @Accept       json
// @Produce      json
// @Router       /comments  [post]
func CommentCreate(app *core.App, router fiber.Router) {
	router.Post("/comments", common.LimiterGuard(app, func(c *fiber.Ctx) error {
		var p ParamsCommentCreate
		if isOK, resp := common.ParamsDecode(c, &p); !isOK {
			return resp
		}

		if strings.TrimSpace(p.Name) == "" {
			return common.RespError(c, 400, i18n.T("{{name}} cannot be empty", Map{"name": i18n.T("Nickname")}))
		}
		if strings.TrimSpace(p.Email) == "" {
			return common.RespError(c, 400, i18n.T("{{name}} cannot be empty", Map{"name": i18n.T("Email")}))
		}

		if !utils.ValidateEmail(p.Email) {
			return common.RespError(c, 400, i18n.T("Invalid {{name}}", Map{"name": i18n.T("Email")}))
		}
		if p.Link != "" && !utils.ValidateURL(p.Link) {
			return common.RespError(c, 400, i18n.T("Invalid {{name}}", Map{"name": i18n.T("Link")}))
		}

		if _, ok, resp := common.CheckSiteExist(app, c, p.SiteName); !ok {
			return resp
		}

		var (
			ip      = c.IP()
			ua      = string(c.Request().Header.UserAgent())
			referer = string(c.Request().Header.Referer())
			isAdmin = common.CheckIsAdminReq(app, c)
		)

		// 允许传入修正后的 UA
		if p.UA != "" {
			ua = p.UA
		}

		// find page
		page := app.Dao().FindCreatePage(p.PageKey, p.PageTitle, p.SiteName)
		if page.Key == "" {
			log.Error("[CommentCreate] FindCreatePage error")
			return common.RespError(c, 500, i18n.T("Comment failed"))
		}

		// check if the user is allowed to comment
		if isAllowed, resp := isAllowComment(app, c, p.Name, p.Email, page.AdminOnly); !isAllowed {
			return resp
		}

		// check reply comment
		var parentComment entity.Comment
		if p.Rid != 0 {
			parentComment = app.Dao().FindComment(p.Rid)
			if parentComment.IsEmpty() {
				return common.RespError(c, 404, i18n.T("{{name}} not found", Map{"name": i18n.T("Parent comment")}))
			}
			if parentComment.PageKey != p.PageKey {
				return common.RespError(c, 400, "Inconsistent with the page_key of the parent comment")
			}
			if !parentComment.IsAllowReply() {
				return common.RespError(c, 400, i18n.T("Cannot reply to this comment"))
			}
		}

		// find user
		isVerified := true
		user, err := common.GetUserByReq(app, c)
		if errors.Is(err, common.ErrTokenNotProvided) {
			// Anonymous user
			isVerified = false
			user, err = app.Dao().FindCreateUser(p.Name, p.Email, p.Link)
			if err != nil {
				log.Error("[CommentCreate] Create user error: ", err)
				return common.RespError(c, 500, i18n.T("Comment failed"))
			}

			// Update user
			user.Link = p.Link
			user.LastIP = ip
			user.LastUA = ua
			user.Name = p.Name // for 若用户修改用户名大小写
			user.Email = p.Email
			app.Dao().UpdateUser(&user)
		} else if err != nil {
			// Login user error
			log.Error("[CommentCreate] Get user error: ", err)
			return common.RespError(c, 500, i18n.T("Comment failed"))
		}

		comment := entity.Comment{
			Content:  p.Content,
			PageKey:  page.Key,
			SiteName: p.SiteName,

			UserID: user.ID,
			IP:     ip,
			UA:     ua,

			Rid:    p.Rid,
			RootID: app.Dao().FindCommentRootID(p.Rid),

			IsPending:   false,
			IsCollapsed: false,
			IsPinned:    false,
			IsVerified:  isVerified,
		}

		// default comment type
		if !isAdmin && app.Conf().Moderator.PendingDefault {
			// 不是管理员评论 && 配置开启评论默认待审
			comment.IsPending = true
		}

		// save to database
		if err := app.Dao().CreateComment(&comment); err != nil {
			log.Error("Save Comment error: ", err)
			return common.RespError(c, 500, i18n.T("Comment failed"))
		}

		// 异步执行
		go func() {
			// Page Update
			if app.Dao().CookPage(&page).URL != "" && page.Title == "" {
				app.Dao().FetchPageFromURL(&page)
			}

			// 垃圾检测
			if !isAdmin { // 忽略检查管理员
				// 同步执行
				if antiSpamService, err := core.AppService[*core.AntiSpamService](app); err == nil {
					antiSpamService.CheckAndBlock(&core.AntiSpamCheckPayload{
						Comment:      &comment,
						ReqReferer:   referer,
						ReqIP:        ip,
						ReqUserAgent: ua,
					})
				} else {
					log.Error("[AntiSpamService] err: ", err)
				}
			}

			// 通知发送
			if notifyService, err := core.AppService[*core.NotifyService](app); err == nil {
				if err := notifyService.Push(&comment, &parentComment); err != nil {
					log.Error("[NotifyService] notify push err: ", err)
				}
			} else {
				log.Error("[NotifyService] err: ", err)
			}
		}()

		cookedComment := app.Dao().CookComment(&comment)
		cookedComment = fetchIPRegionForComment(app, cookedComment)

		return common.RespData(c, ResponseCommentCreate{
			CookedComment: cookedComment,
		})
	}))
}

// Fetch IP Region for Comment
func fetchIPRegionForComment(app *core.App, comment entity.CookedComment) entity.CookedComment {
	if app.Conf().IPRegion.Enabled {
		if ipRegionService, err := core.AppService[*core.IPRegionService](app); err == nil {
			comment.IPRegion = ipRegionService.Query(comment.IP)
		} else {
			log.Error("[IPRegionService] err: ", err)
		}
	}
	return comment
}

func isAllowComment(app *core.App, c *fiber.Ctx, name string, email string, pageAdminOnly bool) (bool, error) {
	// if the user is an admin user or page is admin only
	isAdminUser := app.Dao().IsAdminUserByNameEmail(name, email)
	if isAdminUser || pageAdminOnly {
		// then check has admin access
		if !common.CheckIsAdminReq(app, c) {
			return false, common.RespError(c, 403, i18n.T("Admin access required"), Map{"need_login": true})
		}
	}

	// if token is provided, then check token is valid
	user, err := common.GetUserByReq(app, c)
	if !errors.Is(err, common.ErrTokenNotProvided) && user.IsEmpty() {
		// need_auth_login is a hook for frontend to show login modal (new Auth api)
		return false, common.RespError(c, 401, i18n.T("Login required"), Map{"need_auth_login": true})
	}

	return true, nil
}
