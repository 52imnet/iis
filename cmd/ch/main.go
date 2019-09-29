package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"html/template"
	_ "image/png"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

var m *Manager

func main() {
	var err error
	loadConfig()

	if os.Getenv("IIS_NAME") != "" {
		s := make([]byte, 8)

		keybuf := []byte(config.Key)
		for i := 0; ; i++ {
			rand.Read(s)
			for i := range s {
				s[i] = "abcdefghijklmnopqrstuvwxyz0123456789"[s[i]%36]
			}

			h := hmac.New(sha1.New, keybuf)
			h.Write(s)
			h.Write(keybuf)

			x := h.Sum(nil)

			y := base64.URLEncoding.EncodeToString(x[:4])[:5]
			if (y[0] == 'y' || y[0] == 'Y') &&
				(y[1] == 'm' || y[1] == 'M') &&
				(y[2] == 'o' || y[2] == 'O' || y[2] == '0') &&
				(y[3] == 'u' || y[3] == 'U') &&
				(y[4] == 's' || y[4] == 'S' || y[4] == '5') {
				fmt.Println("\nresult:", string(s))
				break
			}

			if i%1e3 == 0 {
				fmt.Printf("\rprogress: %dk", i/1e3)
			}
		}
	}

	m, err = NewManager("iis.db")
	if err != nil {
		panic(err)
	}

	os.MkdirAll("tmp/logs", 0700)
	logf, err := rotatelogs.New("tmp/logs/access_log.%Y%m%d%H%M", rotatelogs.WithLinkName("tmp/logs/access_log"), rotatelogs.WithMaxAge(7*24*time.Hour))
	logerrf, err := rotatelogs.New("tmp/logs/error_log.%Y%m%d%H%M", rotatelogs.WithLinkName("tmp/logs/error_log"), rotatelogs.WithMaxAge(7*24*time.Hour))
	if err != nil {
		panic(err)
	}

	if config.Key != "0123456789abcdef" {
		log.Println("P R O D U C A T I O N")
		gin.SetMode(gin.ReleaseMode)
		mwLoggerOutput, gin.DefaultErrorWriter = logf, logerrf
	} else {
		mwLoggerOutput, gin.DefaultErrorWriter = io.MultiWriter(logf, os.Stdout), io.MultiWriter(logerrf, os.Stdout)
	}

	log.SetOutput(mwLoggerOutput)
	log.SetFlags(log.Lshortfile | log.Ltime | log.Ldate)

	titles := []string{
		"ofo押金难退，你可以试试“假破产真逼债”，但是不建议 手机发帖  ...2 New	",
		"各省适龄学生参加高考参加率以及其211、985录取率 手机发帖  ...2 New	",
		"猫咪缺铁性贫血 该怎么补 attach_img New	",
		"国产人造肉九月上市 手机发帖  ...2 New	",
		"7天内美国三个枪手的照片排排看。 New	",
		"台风让闻得出别人身上地铁站味道的小布尔乔亚崩溃了  ...23456 New	",
		"为啥龙族吧会变成黑江南的呢？ 手机发帖  ...23	",
		"“你可以不同意我的观点，但是我会捍卫你说话的权利”  ...23	",
		"超员1人被查，司机怒将3岁儿子甩丢出去，准备驾车离开	",
		"突然发现人与人的处境多么奇妙 attach_img	",
		"秀得飞起的高速轮胎，开头无厘头片结尾惊悚片	",
		"太极大师马保国亲戚张麒约战50岁业余摔跤手邓勇	",
		"颜值成第一择偶条件 上海青年婚恋数据曝光  ...234	",
		"咸阳51岁独居男家中去世 一周后被发现  ...2	",
		"工商联：北京取消限购，各区单独设置新能源汽车号牌！	",
		"乌贼娘《诡秘之主》集中讨论帖：scp克苏鲁蒸汽朋克 attach_img 手机发帖 heatlevel  ...23456..447	",
		"脑洞，几个利奇马级别的台风能改变撒哈拉沙漠的地形地貌？	",
		"华为这个次世代地图的饼画的还真有点意思 attach_img  ...23	",
		"【持续更新图片】工作的营地隔壁发生了针对车辆IED炸弹... attach_img 手机发帖  ...234	",
		"这不就是唐僧在车迟国比试的那个运动吗？ 手机发帖  ...2	",
		"朋友家小区楼下的告示，管理人尽力了 手机发帖	",
		"2.4米长眼镜王蛇爬入农家浴室 女主人被吓跑	",
		"力保健也一般呐……	",
		"那多笔记为什么感觉吊着一口气，欠点火候	",
		"深圳1.5亿的房子长什么样子？! 手机发帖  ...2345	",
		"想了解下论坛各位的父母吵架/相处情况 手机发帖  ...234	",
		"求推万元左右的电钢琴 attach_img  ...2	",
		"邮轮答疑帖S2 走咯？上船去咯？ attach_img	",
		"总感觉有点怪，路上老有人找我问路 手机发帖  ...2	",
		"微信表情包的麻将脸太大了，没了感觉 attach_img	",
		"【树洞】爹妈年纪大了开始迷信，真是无解的难题 attach_img  ...2	",
		"T恤设计将港澳归为国家 范思哲道歉：已下架并销毁 手机发帖	",
		"洪阿姨勇气可嘉",
	}

	var last []byte
	for i, t := range titles {
		a := m.NewPost(strconv.Itoa(i)+" ---"+t, "ddd", "zzz", "127.0.0.1", []string{"标签", "Test"})
		log.Println(m.PostPost(a))
		last = a.ID
	}

	for i, t := range titles {
		a := m.NewReply(strconv.Itoa(i)+" ---"+t, "zzz", "127.0.0.1")
		m.PostReply(last, a)
	}

	// nodes := []*driver.Node{}
	//for _, s := range config.Storages {
	//	if s.Name == "" {
	//		panic("empty storage node name")
	//	}
	//	log.Println("[config] load storage:", s.Name)
	//	switch strings.ToLower(s.Type) {
	//	case "dropbox":
	//		nodes = append(nodes, chdropbox.NewNode(s.Name, s))
	//	default:
	//		panic("unknown storage type: " + s.Type)
	//	}
	//}

	//mgr.LoadNodes(nodes)
	//mgr.StartTransferAgent("tmp/transfer.db")
	//cachemgr = cache.New("tmp/cache", config.CacheSize*1024*1024*1024, func(k string) ([]byte, error) {
	//	return mgr.Get(k)
	//})
	//go uploadLocalImages()

	r := gin.New()
	r.Use(gin.Recovery(), gzip.Gzip(gzip.BestSpeed), mwLogger(), mwRenderPerf, mwIPThrot)
	r.SetFuncMap(template.FuncMap{
		"RenderPerf": func() string {
			return fmt.Sprintf("%dms/%dms/%.3fM", survey.render.avg, survey.render.max, float64(survey.written)/1024/1024)
		},
	})
	r.LoadHTMLGlob("template/*.html")
	r.Static("/s/", "template")
	r.Handle("GET", "/", func(g *gin.Context) { g.HTML(200, "home.html", struct{ Home template.HTML }{}) })
	r.Handle("GET", "/vec", makeHandleMainView('v'))
	r.Handle("GET", "/p/:parent", handleRepliesView)
	r.Handle("GET", "/tag/:tag", makeHandleMainView('t'))
	r.Handle("GET", "/tags", handleTags)
	r.Handle("GET", "/id/:id", makeHandleMainView('a'))
	r.Handle("GET", "/new", handleNewPostView)
	r.Handle("GET", "/stat", handleCurrentStat)
	r.Handle("GET", "/edit/:id", handleEditPostView)

	r.Handle("POST", "/new", handleNewPostAction)
	r.Handle("POST", "/reply", handleNewReplyAction)
	r.Handle("POST", "/edit", handleEditPostAction)
	r.Handle("POST", "/delete", handleDeletePostAction)

	r.Handle("GET", "/cookie", handleCookie)
	r.Handle("POST", "/cookie", handleCookie)

	if config.Domain == "" {
		log.Fatal(r.Run(":5010"))
	} else {
		go func() {
			time.Sleep(time.Second)
			log.Println("plain server started on :80")
			log.Fatal(r.Run(":80"))
		}()
		log.Fatal(autotls.Run(r, config.Domain))
	}
}

func handleCookie(g *gin.Context) {
	if g.Request.Method == "GET" {
		id, _ := g.Cookie("id")
		g.HTML(200, "cookie.html", struct{ ID string }{id})
		return
	}
	if id := g.PostForm("id"); g.PostForm("clear") != "" || id == "" {
		g.SetCookie("id", "", -1, "", "", false, false)
	} else if g.PostForm("notify") != "" {
		g.Redirect(302, "/inbox/"+id)
		return
	} else {
		g.SetCookie("id", id, 86400*365, "", "", false, false)
	}
	g.Redirect(302, "/vec")
}

func makeHandleMainView(t byte) func(g *gin.Context) {
	return func(g *gin.Context) {
		var bkName, partKey []byte
		var pl ArticlesTimelineView
		var err error
		var more bool

		switch t {
		case 't':
			pl.SearchTerm, pl.Type = g.Param("tag"), "tag"
			bkName, partKey = bkAuthorTag, []byte("#"+pl.SearchTerm)
		case 'a':
			pl.SearchTerm, pl.Type = g.Param("id"), "id"
			bkName, partKey = bkAuthorTag, []byte(pl.SearchTerm)
		default:
			bkName = bkPost
		}

		next, dir := parseCursor(g.Query("n"))
		if dir == "prev" {
			pl.Articles, more, pl.TotalCount, err = m.FindPosts('a', bkName, partKey, next, int(config.PostsPerPage))
			pl.NoPrev = !more
		} else {
			pl.Articles, more, pl.TotalCount, err = m.FindPosts('d', bkName, partKey, next, int(config.PostsPerPage))
			pl.NoPrev = next == nil
		}

		if err != nil {
			errorPage(500, "INTERNAL: "+err.Error(), g)
			return
		}

		for _, a := range pl.Articles {
			a.SearchTerm = pl.SearchTerm
		}

		if len(pl.Articles) > 0 {
			pl.Next = objectIDToDisplayID(incBytes(pl.Articles[len(pl.Articles)-1].ID, -1))
			pl.Prev = objectIDToDisplayID(incBytes(pl.Articles[0].ID, 1))
			pl.Title = fmt.Sprintf("%s ~ %s", pl.Articles[0].CreateTimeString(true), pl.Articles[len(pl.Articles)-1].CreateTimeString(true))
		}

		g.HTML(200, "index.html", pl)
	}
}

func handleRepliesView(g *gin.Context) {
	var pl = ArticleRepliesView{ShowIP: isAdmin(g)}
	var err error

	pl.ParentArticle, err = m.GetArticle(displayIDToObejctID(g.Param("parent")))
	if err != nil || pl.ParentArticle.ID == nil {
		errorPage(404, "NOT FOUND", g)
		log.Println(pl.ParentArticle, err)
		return
	}

	pl.ReplyView = generateNewReplyView(pl.ParentArticle.ID, g)
	pl.CurPage, _ = strconv.Atoi(g.Query("p"))
	pl.TotalPages = int(math.Ceil(float64(len(pl.ParentArticle.Replies)) / float64(config.PostsPerPage)))

	incrCounter(g, pl.ParentArticle.ID)

	switch pl.CurPage {
	case 0:
		pl.CurPage = 1
	case -1:
		pl.CurPage = pl.TotalPages
	default:
		pl.CurPage = intmin(pl.CurPage, pl.TotalPages)
	}

	if pl.TotalPages > 0 {
		start := intmin(len(pl.ParentArticle.Replies), (pl.CurPage-1)*config.PostsPerPage)
		end := intmin(len(pl.ParentArticle.Replies), pl.CurPage*config.PostsPerPage)

		pl.Articles = mgetReplies(pl.ParentArticle.ID, pl.ParentArticle.Replies[start:end])
		pl.Pages = make([]int, 0, pl.TotalPages)

		for i := pl.CurPage - 3; i <= pl.CurPage+3; i++ {
			if i > 0 && i <= pl.TotalPages {
				pl.Pages = append(pl.Pages, i)
			}
		}

		for len(pl.Pages) < 7 {
			if last := pl.Pages[len(pl.Pages)-1]; last+1 <= pl.TotalPages {
				pl.Pages = append(pl.Pages, last+1)
			} else {
				break
			}
		}

		for len(pl.Pages) < 7 {
			if first := pl.Pages[0]; first-1 > 0 {
				pl.Pages = append([]int{first - 1}, pl.Pages...)
			} else {
				break
			}
		}
	}

	g.HTML(200, "post.html", pl)
}
