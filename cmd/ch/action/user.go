package action

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"log"
	"strings"
	"time"

	"github.com/coyove/iis/cmd/ch/config"
	"github.com/coyove/iis/cmd/ch/ident"
	mv "github.com/coyove/iis/cmd/ch/model"
	"github.com/gin-gonic/gin"
)

func User(g *gin.Context) {

	var (
		ip        = hashIP(g)
		username  = sanUsername(g.PostForm("username"))
		email     = mv.SoftTrunc(g.PostForm("email"), 64)
		password  = mv.SoftTrunc(g.PostForm("password"), 32)
		password2 = mv.SoftTrunc(g.PostForm("password2"), 32)
		mth       = g.PostForm("method")
		redir     = func(a, b string) {
			if mth == "login" && a == "error" {
				a = "login-error"
			}
			q := EncodeQuery(a, b, "username", username, "email", email, "password", ident.MakeTempToken(password))
			if mth == "login" {
				u := g.Request.Referer()
				if idx := strings.Index(u, "?"); idx > -1 {
					u = u[:idx]
				}
				g.Redirect(302, u+q)
			} else {
				g.Redirect(302, "/user"+q)
			}
		}
		u *mv.User
	)

	if g.PostForm("cancel") != "" {
		redir("", "")
		return
	}

	if mth == "logout" {
		u = &mv.User{}
		goto SKIP
	}

	if mth == "update-email" {
		if ret := checkToken(g); ret != "" {
			redir("error", ret)
			return
		}
		user, _ := g.Get("user")
		if user == nil {
			redir("error", "guard/id-not-existed")
			return
		}
		u = user.(*mv.User)
		u.Email = email
		goto SKIP
	}

	if len(username) < 3 {
		redir("error", "id/too-short")
		return
	}

SKIP:
	m.LockUserID(username)
	defer m.UnlockUserID(username)

	pwdHash := hmac.New(sha256.New, config.Cfg.KeyBytes)
	pwdHash.Write([]byte(password))

	switch mth {
	case "signup":
		if ret := checkCaptcha(g); ret != "" {
			redir("error", ret)
			return
		}

		if len(password) == 0 || password != password2 {
			redir("error", "password/invalid-too-short")
			return
		}

		if u, err := m.GetUser(username); err == nil && u.ID == username {
			redir("error", "id/already-existed")
			return
		}

		u = &mv.User{
			ID:           username,
			Session:      genSession(),
			Email:        email,
			PasswordHash: pwdHash.Sum(nil),
			SignupIP:     ip,
			Signup:       time.Now(),
			LoginIP:      ip,
			Login:        time.Now(),
		}
	case "login":

		if ret := checkToken(g); ret != "" {
			redir("error", ret)
			return
		}
		u, _ = m.GetUser(username)
		if u == nil {
			redir("error", "guard/id-not-existed")
			return
		}

		if !bytes.Equal(u.PasswordHash, pwdHash.Sum(nil)) {
			redir("error", "internal/error")
			return
		}
		u.Session = genSession()
		u.Login = time.Now()
		u.LoginIP = ip
	}

	if err := m.SetUser(u); err != nil {
		log.Println(u, err)
		redir("error", "internal/error")
		return
	}

	g.SetCookie("id", mv.MakeUserToken(u), 365*86400, "", "", false, false)
	redir("", "")
}