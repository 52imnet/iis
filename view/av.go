package view

import (
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/coyove/iis/common"
	"github.com/coyove/iis/dal"
	"github.com/coyove/iis/ik"
	"github.com/coyove/iis/model"
)

type ArticleView struct {
	ID            string
	Others        []*ArticleView
	Parent        *ArticleView
	Author        *model.User
	You           *model.User
	Cmd           string
	Replies       int
	Likes         int
	ReplyLockMode byte
	Liked         bool
	NSFW          bool
	NoAvatar      bool
	GreyOutReply  bool
	AlsoReply     bool
	StickOnTop    bool
	Content       string
	ContentHTML   template.HTML
	Media         string
	MediaType     string
	History       string
	CreateTime    time.Time
}

const (
	_ uint64 = 1 << iota
	_NoMoreParent
	_ShowAuthorAvatar
	_GreyOutReply
	_NoCluster
)

func NewTopArticleView(a *model.Article, you *model.User) (av ArticleView) {
	av.from(a, 0, you)
	return
}

func NewReplyArticleView(a *model.Article, you *model.User) (av ArticleView) {
	av.from(a, _NoMoreParent|_ShowAuthorAvatar, you)
	return
}

func (a *ArticleView) from(a2 *model.Article, opt uint64, u *model.User) *ArticleView {
	if a2 == nil {
		return a
	}

	a.ID = a2.ID
	a.Replies = int(a2.Replies)
	a.Likes = int(a2.Likes)
	a.ReplyLockMode = a2.ReplyLockMode
	a.NSFW = a2.NSFW
	a.StickOnTop = a2.StickOnTop()
	a.Cmd = string(a2.Cmd)
	a.CreateTime = a2.CreateTime
	a.History = a2.History
	a.Author, _ = dal.GetUser(a2.Author)
	if a.Author == nil {
		a.Author = (&model.User{ID: a2.Author}).SetInvalid()
	}
	a.You = u
	if a.You == nil {
		a.You = &model.User{}
	} else {
		a.Liked = dal.IsLiking(u.ID, a2.ID)
	}

	if p := strings.SplitN(a2.Media, ":", 2); len(p) == 2 {
		a.MediaType, a.Media = p[0], p[1]
		switch a.MediaType {
		case "IMG":
			a.Media = common.Cfg.MediaDomain + "/i/" + strings.TrimPrefix(a.Media, "LOCAL:") + ".jpg"
		}
	}

	// if img := common.ExtractFirstImage(a2.Content); img != "" && a2.Media == "" {
	// 	a.MediaType, a.Media = "IMG", img
	// }

	a.Content = a2.Content
	a.ContentHTML = a2.ContentHTML()

	if a2.Parent != "" {
		a.Parent = &ArticleView{}
		if opt&_NoMoreParent == 0 {
			p, _ := dal.GetArticle(a2.Parent)
			a.Parent.from(p, opt&(^_GreyOutReply)|_NoMoreParent, u)
		}
	}

	a.GreyOutReply = opt&_GreyOutReply > 0
	a.NoAvatar = opt&_NoMoreParent > 0
	if opt&_ShowAuthorAvatar > 0 {
		a.NoAvatar = false
	}

	switch a2.Cmd {
	case model.CmdInboxReply, model.CmdInboxMention:
		p, _ := dal.GetArticle(a2.Extras["article_id"])
		if p == nil {
			return a
		}

		a.from(p, opt, u)
		a.Cmd = string(a2.Cmd)
	case model.CmdInboxFwAccepted:
		dummy := &model.Article{
			ID:         ik.NewGeneralID().String(),
			CreateTime: a2.CreateTime,
			Author:     a2.Extras["from"],
		}
		a.from(dummy, opt, u)
		a.Cmd = model.CmdInboxFwAccepted
	case model.CmdInboxLike:
		p, _ := dal.GetArticle(a2.Extras["article_id"])

		if p == nil {
			return a
		}

		dummy := &model.Article{
			ID:         ik.NewGeneralID().String(),
			CreateTime: p.CreateTime,
			Cmd:        model.CmdInboxLike,
			Author:     a2.Extras["from"],
			Parent:     p.ID,
		}
		a.from(dummy, opt, u)
	}

	return a
}

func fromMultiple(a *[]ArticleView, a2 []*model.Article, opt uint64, u *model.User) {
	*a = make([]ArticleView, len(a2))

	lookup := map[string]*ArticleView{}
	dedup := map[string]bool{}
	cluster := map[string][]string{}

	for i, v := range a2 {
		(*a)[i].from(v, opt, u)
		lookup[(*a)[i].ID] = &(*a)[i]
	}

	// First pass, connect continous replies (r1 -> r2 -> ... -> first_post) into one post
	for i, v := range *a {
		if v.Parent != nil {
			cluster[v.Parent.ID] = append(cluster[v.Parent.ID], v.ID) // will be used in second pass

			if lookup[v.Parent.ID] != nil {
				p := lookup[v.Parent.ID]
				p.NoAvatar = true
				(*a)[i].Parent = p
				dedup[p.ID] = true
			}
		}
	}

	if opt&_NoCluster == 0 {
		// Second pass, connect clustered replies (r1 -> p, r2 -> p ...., rn -> p) into one post
	NEXT:
		for _, rids := range cluster {
			if len(rids) == 1 {
				continue
			}
			for _, rid := range rids {
				if dedup[rid] {
					continue NEXT
				}
			}
			// All posts are visible
			r := lookup[rids[0]]
			for i := 1; i < len(rids); i++ {
				ri := lookup[rids[i]]
				if ri == nil {
					log.Println("[ClusterReplies] nil:", rids[i])
					continue
				}
				ri.Parent = nil
				ri.Others = nil
				ri.AlsoReply = true
				r.Others = append(r.Others, ri)
				dedup[rids[i]] = true
			}
		}
	}

	newa := make([]ArticleView, 0, len(*a))
	for _, v := range *a {
		if dedup[v.ID] {
			continue
		}
		newa = append(newa, v)
	}
	*a = newa
}
