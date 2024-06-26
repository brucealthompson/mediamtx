package hls

import (
	_ "embed"
	"errors"
	"net"
	"net/http"
	gopath "path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bluenviron/mediamtx/internal/auth"
	"github.com/bluenviron/mediamtx/internal/conf"
	"github.com/bluenviron/mediamtx/internal/defs"
	"github.com/bluenviron/mediamtx/internal/logger"
	"github.com/bluenviron/mediamtx/internal/protocols/httpp"
	"github.com/bluenviron/mediamtx/internal/restrictnetwork"
)

//go:generate go run ./hlsjsdownloader

//go:embed index.html
var hlsIndex []byte

//nolint:typecheck
//go:embed hls.min.js
var hlsMinJS []byte

func mergePathAndQuery(path string, rawQuery string) string {
	res := path
	if rawQuery != "" {
		res += "?" + rawQuery
	}
	return res
}

type httpServer struct {
	address        string
	encryption     bool
	serverKey      string
	serverCert     string
	allowOrigin    string
	trustedProxies conf.IPNetworks
	readTimeout    conf.StringDuration
	pathManager    serverPathManager
	parent         *Server

	inner *httpp.WrappedServer
}

func (s *httpServer) initialize() error {
	router := gin.New()
	gin.SetMode(gin.DebugMode)
	router.SetTrustedProxies(s.trustedProxies.ToTrustedProxies()) //nolint:errcheck
	router.HEAD("/version", func(c *gin.Context) {
		cameras := s.pathManager.GetPaths()
		camerasvalue := ""
		for index, camera := range cameras {
			_, cameraup := s.parent.muxers[camera]
			cameraupstring := "down"
			if cameraup {
				cameraupstring = "up"
			}
			if index > 0 {
				camerasvalue += ","
			}
			camerasvalue += camera + ":" + cameraupstring
		}
		c.Writer.Header().Add("cameras", camerasvalue)
		c.Writer.Header().Set("mediamtx", "Version 1.0")
	})
	router.LoadHTMLGlob("web/templates/*")
	router.StaticFS("/static", http.Dir("web/static"))
	router.GET("/", func(c *gin.Context) {
		cameras := s.pathManager.GetPaths()
		activecamera := ""
		query := c.Request.URL.Query()
		camera, ok := query["cameraname"]
		if ok && (len(camera) == 1) && (len(camera[0]) > 0) {
			activecamera = camera[0]
		} else if len(cameras) > 0 {
			activecamera = cameras[0]
		}
		//sort.Strings(all)
		if len(activecamera) > 0 {
			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"suuid":    activecamera,
				"suuidMap": cameras,
				"version":  time.Now().String(),
			})
		}
	})
	router.NoRoute(s.onRequest)

	network, address := restrictnetwork.Restrict("tcp", s.address)

	s.inner = &httpp.WrappedServer{
		Network:     network,
		Address:     address,
		ReadTimeout: time.Duration(s.readTimeout),
		Encryption:  s.encryption,
		ServerCert:  s.serverCert,
		ServerKey:   s.serverKey,
		Handler:     router,
		Parent:      s,
	}
	err := s.inner.Initialize()
	if err != nil {
		return err
	}

	return nil
}

// Log implements logger.Writer.
func (s *httpServer) Log(level logger.Level, format string, args ...interface{}) {
	s.parent.Log(level, format, args...)
}

func (s *httpServer) close() {
	s.inner.Close()
}

func (s *httpServer) onRequest(ctx *gin.Context) {
	ctx.Writer.Header().Set("Access-Control-Allow-Origin", s.allowOrigin)
	ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

	switch ctx.Request.Method {
	case http.MethodOptions:
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Range")
		ctx.Writer.WriteHeader(http.StatusNoContent)
		return

	case http.MethodGet:

	default:
		return
	}

	// remove leading prefix
	pa := ctx.Request.URL.Path[1:]

	var dir string
	var fname string

	switch {
	case strings.HasSuffix(pa, "/hls.min.js"):
		ctx.Writer.Header().Set("Cache-Control", "max-age=3600")
		ctx.Writer.Header().Set("Content-Type", "application/javascript")
		ctx.Writer.WriteHeader(http.StatusOK)
		ctx.Writer.Write(hlsMinJS)
		return

	case pa == "", pa == "favicon.ico", strings.HasSuffix(pa, "/hls.min.js.map"):
		return

	case strings.HasSuffix(pa, ".m3u8") ||
		strings.HasSuffix(pa, ".ts") ||
		strings.HasSuffix(pa, ".mp4") ||
		strings.HasSuffix(pa, ".mp"):
		dir, fname = gopath.Dir(pa), gopath.Base(pa)

		if strings.HasSuffix(fname, ".mp") {
			fname += "4"
		}

	default:
		dir, fname = pa, ""

		if !strings.HasSuffix(dir, "/") {
			ctx.Writer.Header().Set("Location", mergePathAndQuery(ctx.Request.URL.Path+"/", ctx.Request.URL.RawQuery))
			ctx.Writer.WriteHeader(http.StatusMovedPermanently)
			return
		}
	}

	dir = strings.TrimSuffix(dir, "/")
	if dir == "" {
		return
	}

	user, pass, hasCredentials := ctx.Request.BasicAuth()

	pathConf, err := s.pathManager.FindPathConf(defs.PathFindPathConfReq{
		AccessRequest: defs.PathAccessRequest{
			Name:    dir,
			Query:   ctx.Request.URL.RawQuery,
			Publish: false,
			IP:      net.ParseIP(ctx.ClientIP()),
			User:    user,
			Pass:    pass,
			Proto:   auth.ProtocolHLS,
		},
	})
	if err != nil {
		var terr auth.Error
		if errors.As(err, &terr) {
			if !hasCredentials {
				ctx.Header("WWW-Authenticate", `Basic realm="mediamtx"`)
				ctx.Writer.WriteHeader(http.StatusUnauthorized)
				return
			}

			s.Log(logger.Info, "connection %v failed to authenticate: %v", httpp.RemoteAddr(ctx), terr.Message)

			// wait some seconds to mitigate brute force attacks
			<-time.After(auth.PauseAfterError)

			ctx.Writer.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx.Writer.WriteHeader(http.StatusNotFound)
		return
	}

	switch fname {
	case "":
		ctx.Writer.Header().Set("Cache-Control", "max-age=3600")
		ctx.Writer.Header().Set("Content-Type", "text/html")
		ctx.Writer.WriteHeader(http.StatusOK)
		ctx.Writer.Write(hlsIndex)

	default:
		mux, err := s.parent.getMuxer(serverGetMuxerReq{
			path:           dir,
			remoteAddr:     httpp.RemoteAddr(ctx),
			query:          ctx.Request.URL.RawQuery,
			sourceOnDemand: pathConf.SourceOnDemand,
		})
		if err != nil {
			ctx.Writer.WriteHeader(http.StatusNotFound)
			return
		}

		mi := mux.getInstance()
		if mi == nil {
			ctx.Writer.WriteHeader(http.StatusNotFound)
			return
		}

		ctx.Request.URL.Path = fname
		mi.handleRequest(ctx)
	}
}
