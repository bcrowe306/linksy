package handler

import (
	"errors"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/bcrowe306/linksy/internal/api/middleware"
	"github.com/bcrowe306/linksy/internal/auth"
	"github.com/bcrowe306/linksy/internal/service"
)

// ----------------- Admin re-authentication (AuthHandler: needs WorkOS) -----------------

func (a *AuthHandler) AdminStatus(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	st, err := a.svc.AdminStatus(ac.Session.ID, ac.UserID)
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"is_admin": st.IsAdmin, "elevated": st.Elevated, "elevated_until": st.ElevatedUntil,
		"methods": fiber.Map{"password": a.workos.Enabled() || a.devLoginEnabled(), "email_code": st.EmailCode}})
}

type adminReauthRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

// AdminReauth unlocks the admin console with the account password or an emailed code.
func (a *AuthHandler) AdminReauth(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	var req adminReauthRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	ip := middleware.ClientIP(c)
	var (
		until *time.Time
		err   error
	)
	switch {
	case req.Code != "":
		until, err = a.svc.VerifyAdminCode(ac.Session.ID, ac.UserID, req.Code, ip)
	case req.Password != "":
		u, uerr := a.svc.GetUserProfile(ac.UserID)
		if uerr != nil || !u.IsPlatformAdmin {
			return fail(c, service.ErrNotPlatformAdmin)
		}
		switch {
		case a.workos.Enabled():
			ctx, cancel := a.ctx()
			defer cancel()
			res, werr := a.workos.AuthenticateWithPassword(ctx, u.Email, req.Password, ip, c.Get("User-Agent"))
			var we *auth.WorkOSError
			if werr != nil && !(errors.As(werr, &we) && we.IsCode("email_verification_required")) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "That password is incorrect", "code": "invalid_password"})
			}
			if res != nil && res.User.ID != u.ID {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "That password is incorrect", "code": "invalid_password"})
			}
		case a.devLoginEnabled():
			// Local development has no identity provider; any password confirms.
		default:
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Password sign-in is not configured; use an email code"})
		}
		until, err = a.svc.ElevateAdmin(ac.Session.ID, ac.UserID, ip, "password")
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Enter your password or the emailed code"})
	}
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"elevated": true, "elevated_until": until})
}

func (a *AuthHandler) AdminSendCode(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	if err := a.svc.SendAdminCode(ac.Session.ID, ac.UserID); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"sent": true})
}

func (a *AuthHandler) AdminLeave(c fiber.Ctx) error {
	ac := middleware.GetAuth(c)
	if err := a.svc.LeaveAdmin(ac.Session.ID); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"elevated": false})
}

// ----------------- Admin console API -----------------

func (h *Handler) AdminMetrics(c fiber.Ctx) error {
	m, err := h.svc.PlatformMetrics(queryInt(c, "days", 30))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(m)
}

func (h *Handler) AdminListTeams(c fiber.Ctx) error {
	list, err := h.svc.AdminListTeams(c.Query("q"), queryInt(c, "limit", 50), queryInt(c, "offset", 0))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"workspaces": list})
}

func (h *Handler) AdminGetTeam(c fiber.Ctx) error {
	d, err := h.svc.AdminGetTeam(c.Params("id"))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(d)
}

type adminTeamAction struct {
	Days   int    `json:"days"`
	Comped *bool  `json:"comped"`
	Reason string `json:"reason"`
}

// AdminUpdateTeam handles POST /admin/workspaces/:id/{trial|comp|suspend|unsuspend}.
func (h *Handler) AdminUpdateTeam(c fiber.Ctx) error {
	var req adminTeamAction
	if err := bindOptional(c, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	a, id := middleware.Admin(c), c.Params("id")
	var (
		err  error
		team any
	)
	switch c.Params("action") {
	case "trial":
		team, err = h.svc.AdminExtendTrial(a, id, req.Days)
	case "comp":
		if req.Comped == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "comped is required"})
		}
		team, err = h.svc.AdminSetComped(a, id, *req.Comped)
	case "suspend":
		if strings.TrimSpace(req.Reason) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Give a reason for the suspension"})
		}
		team, err = h.svc.AdminSuspendTeam(a, id, req.Reason)
	case "unsuspend":
		team, err = h.svc.AdminSuspendTeam(a, id, "")
	default:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Unknown action"})
	}
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"workspace": team})
}

func (h *Handler) AdminListUsers(c fiber.Ctx) error {
	list, err := h.svc.AdminListUsers(c.Query("q"), queryInt(c, "limit", 50), queryInt(c, "offset", 0))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"users": list})
}

func (h *Handler) AdminGetUser(c fiber.Ctx) error {
	d, err := h.svc.AdminGetUser(c.Params("id"))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(d)
}

type adminUserAction struct {
	Reason string `json:"reason"`
	Admin  *bool  `json:"admin"`
}

// AdminUpdateUser handles POST /admin/users/:id/{suspend|unsuspend|admin}.
func (h *Handler) AdminUpdateUser(c fiber.Ctx) error {
	var req adminUserAction
	if err := bindOptional(c, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	a, id := middleware.Admin(c), c.Params("id")
	var (
		err  error
		user any
	)
	switch c.Params("action") {
	case "suspend":
		if strings.TrimSpace(req.Reason) == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Give a reason for the suspension"})
		}
		user, err = h.svc.AdminSuspendUser(a, id, req.Reason)
	case "unsuspend":
		user, err = h.svc.AdminSuspendUser(a, id, "")
	case "admin":
		if req.Admin == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "admin is required"})
		}
		user, err = h.svc.AdminSetPlatformAdmin(a, id, *req.Admin)
	default:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Unknown action"})
	}
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"user": user})
}

func (h *Handler) AdminListCodes(c fiber.Ctx) error {
	codes, err := h.svc.AdminListCodes()
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(codes)
}

func (h *Handler) AdminCreateCheckoutCode(c fiber.Ctx) error {
	var req service.NewCheckoutCode
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	pc, err := h.svc.AdminCreateCheckoutCode(middleware.Admin(c), req)
	if err != nil {
		return fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(pc)
}

type codeActiveRequest struct {
	Active bool   `json:"active"`
	Code   string `json:"code"` // display label for the audit log
}

func (h *Handler) AdminSetCheckoutCode(c fiber.Ctx) error {
	var req codeActiveRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := h.svc.AdminSetCheckoutCodeActive(middleware.Admin(c), c.Params("id"), req.Code, req.Active); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"active": req.Active})
}

func (h *Handler) AdminCreateTrialCode(c fiber.Ctx) error {
	var req service.NewTrialCode
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	tc, err := h.svc.AdminCreateTrialCode(middleware.Admin(c), req)
	if err != nil {
		return fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(tc)
}

func (h *Handler) AdminSetTrialCode(c fiber.Ctx) error {
	var req codeActiveRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := h.svc.AdminSetTrialCodeActive(middleware.Admin(c), c.Params("code"), req.Active); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"active": req.Active})
}

func (h *Handler) AdminSearchAssets(c fiber.Ctx) error {
	list, err := h.svc.AdminSearchAssets(c.Query("q"))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"assets": list})
}

type moderateRequest struct {
	Reason string `json:"reason"`
}

// AdminModerate handles POST /admin/moderation/:type/:id/{disable|restore}.
func (h *Handler) AdminModerate(c fiber.Ctx) error {
	var req moderateRequest
	if err := bindOptional(c, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	reason := ""
	switch c.Params("action") {
	case "disable":
		if reason = strings.TrimSpace(req.Reason); reason == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Give a reason (shown in the audit log)"})
		}
	case "restore":
	default:
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Unknown action"})
	}
	if err := h.svc.AdminModerate(middleware.Admin(c), c.Params("type"), c.Params("id"), reason); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"disabled": reason != ""})
}

func (h *Handler) AdminListBlocked(c fiber.Ctx) error {
	list, err := h.svc.AdminListBlocked()
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"domains": list})
}

type blockRequest struct {
	Domain string `json:"domain"`
	Reason string `json:"reason"`
}

func (h *Handler) AdminBlockDomain(c fiber.Ctx) error {
	var req blockRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	d, err := h.svc.AdminBlockDomain(middleware.Admin(c), req.Domain, req.Reason)
	if err != nil {
		return fail(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(d)
}

func (h *Handler) AdminUnblockDomain(c fiber.Ctx) error {
	if err := h.svc.AdminUnblockDomain(middleware.Admin(c), c.Params("domain")); err != nil {
		return fail(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) AdminListReports(c fiber.Ctx) error {
	list, err := h.svc.AdminListReports(c.Query("status", "open"))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"reports": list})
}

type resolveRequest struct {
	Action string `json:"action"` // disable | dismiss
}

func (h *Handler) AdminResolveReport(c fiber.Ctx) error {
	var req resolveRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := h.svc.AdminResolveReport(middleware.Admin(c), c.Params("id"), req.Action); err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"resolved": true})
}

func (h *Handler) AdminAudit(c fiber.Ctx) error {
	list, err := h.svc.AdminListAudit(c.Query("q"), queryInt(c, "limit", 50), queryInt(c, "offset", 0))
	if err != nil {
		return fail(c, err)
	}
	return c.JSON(fiber.Map{"events": list})
}

// ----------------- Trial code redemption (workspace billing) -----------------

type redeemRequest struct {
	Code string `json:"code"`
}

func (h *Handler) RedeemTrialCode(c fiber.Ctx) error {
	var req redeemRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	b, err := h.svc.RedeemTrialCode(middleware.Actor(c), req.Code)
	if err != nil {
		return billingCodeFail(c, err)
	}
	return c.JSON(b)
}

// ----------------- Public abuse reports -----------------

var reportReasonLabels = [][2]string{
	{"phishing", "Phishing or stealing passwords"}, {"scam", "Scam or fraud"}, {"malware", "Malware or unwanted software"},
	{"spam", "Spam"}, {"illegal", "Illegal content"}, {"other", "Something else"},
}

// ReportPage serves the public "report abuse" form (GET /report).
func (h *Handler) ReportPage(c fiber.Ctx) error {
	return h.renderReportPage(c, service.AbuseReportInput{URL: c.Query("u")}, "", false, fiber.StatusOK)
}

// SubmitReport accepts the form (POST /report) or JSON (POST /api/v1/report).
func (h *Handler) SubmitReport(c fiber.Ctx) error {
	var in service.AbuseReportInput
	if err := c.Bind().Body(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	err := h.svc.SubmitAbuseReport(in, middleware.ClientIP(c))
	if strings.HasPrefix(c.Path(), "/api/") {
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"received": true})
	}
	if err != nil {
		return h.renderReportPage(c, in, err.Error(), false, fiber.StatusBadRequest)
	}
	return h.renderReportPage(c, service.AbuseReportInput{}, "", true, fiber.StatusOK)
}

func (h *Handler) renderReportPage(c fiber.Ctx, in service.AbuseReportInput, errMsg string, done bool, status int) error {
	var body string
	if done {
		body = `<h1>Thanks for letting us know</h1><p>Our team reviews every report. If the link or QR code breaks our rules, we'll disable it.</p><a class="btn" href="/">Go to Lynqr</a>`
	} else {
		opts := ""
		for _, r := range reportReasonLabels {
			sel := ""
			if in.Reason == r[0] {
				sel = " selected"
			}
			opts += fmt.Sprintf(`<option value="%s"%s>%s</option>`, r[0], sel, r[1])
		}
		errHTML := ""
		if errMsg != "" {
			errHTML = `<p class="err" role="alert">` + html.EscapeString(errMsg) + `</p>`
		}
		body = fmt.Sprintf(`<h1>Report a link or QR code</h1><p>Tell us about a Lynqr short link or QR code that leads somewhere harmful.</p>%s
<form method="POST" action="/report">
<label for="u">Lynqr link or QR address</label><input id="u" name="url" required maxlength="500" placeholder="https://lynqr.app/s/abc123" value="%s">
<label for="r">What's wrong?</label><select id="r" name="reason" required><option value="">Choose a reason</option>%s</select>
<label for="d">Details (optional)</label><textarea id="d" name="details" maxlength="2000" rows="3" placeholder="Where did you see it? What happened?">%s</textarea>
<label for="e">Your email (optional)</label><input id="e" name="email" type="email" maxlength="254" placeholder="So we can follow up" value="%s">
<button type="submit">Send report</button></form>`,
			errHTML, html.EscapeString(in.URL), opts, html.EscapeString(in.Details), html.EscapeString(in.Email))
	}
	page := `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex"><title>Report abuse | Lynqr</title><style>
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;font-family:system-ui,-apple-system,'Segoe UI',sans-serif;background:#0c0c0e;color:#f4f4f6;padding:24px}
.card{max-width:440px;width:100%;background:#16161a;border:1px solid #2c2c32;border-radius:20px;padding:32px}
h1{font-size:20px;margin:0 0 6px}p{color:#acacb4;font-size:14px;margin:0 0 20px;line-height:1.5}
label{display:block;font-size:13px;font-weight:600;margin:14px 0 6px}
input,select,textarea{width:100%;padding:11px 13px;border-radius:12px;border:1px solid #3c3c44;background:#0c0c0e;color:#f4f4f6;font:inherit;font-size:14px;outline:none}
input:focus,select:focus,textarea:focus{border-color:#6366f1;box-shadow:0 0 0 4px rgba(99,102,241,.2)}
button,.btn{display:block;text-align:center;text-decoration:none;margin-top:18px;width:100%;padding:12px;border:0;border-radius:12px;background:linear-gradient(#6366f1,#4f46e5);color:#fff;font-weight:600;font-size:15px;cursor:pointer}
.err{color:#fb7185;font-size:13px;margin:0 0 4px}
</style></head><body><div class="card">` + body + `</div></body></html>`
	c.Set("Cache-Control", "no-store")
	return c.Status(status).Type("html").SendString(page)
}

// renderDisabled is shown for links and QR codes Lynqr disabled for abuse.
func (h *Handler) renderDisabled(c fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	c.Set("X-Robots-Tag", "noindex, nofollow")
	return h.renderBrandedError(c, "This Link Has Been Disabled",
		"Lynqr disabled this link or QR code because it broke our acceptable use policy. Don't enter any personal information on pages it led to.", fiber.StatusGone)
}

// gate renders the right page when a workspace can't serve visitors.
// It returns true when a response was written.
func (h *Handler) gate(c fiber.Ctx, teamID string) (bool, error) {
	readOnly, suspended := h.svc.TeamAccess(teamID)
	switch {
	case suspended:
		return true, h.renderDisabled(c)
	case readOnly:
		return true, h.renderInactiveWorkspace(c)
	}
	return false, nil
}

// bindOptional parses a JSON body when one was sent (action endpoints may have none).
func bindOptional(c fiber.Ctx, v any) error {
	if len(c.Body()) == 0 {
		return nil
	}
	return c.Bind().Body(v)
}
