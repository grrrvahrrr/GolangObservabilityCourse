package openapichi

import (
	"CourseWork/internal/apichi"
	"embed"
	"encoding/json"
	"html/template"
	"net"
	"net/http"
	"runtime/debug"

	chiprometheus "github.com/766b/chi-prometheus"
	"github.com/opentracing/opentracing-go"
	tracelog "github.com/opentracing/opentracing-go/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

//go:embed pages
var tpls embed.FS

type OpenApiChi struct {
	*chi.Mux
	hs     *apichi.Handlers
	m      *BitmeMetrics
	log    *log.Logger
	tracer opentracing.Tracer
}

type PageVars struct {
	ShortURL string
	AdminURL string
	FullURL  string
	Data     string
	IPData   string
}

func NewOpenApiRouter(hs *apichi.Handlers, m *BitmeMetrics, l *log.Logger, tracer opentracing.Tracer) *OpenApiChi {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	//Prometheus middleware for chi
	r.Use(chiprometheus.NewMiddleware("bitme"))

	ret := &OpenApiChi{
		hs:     hs,
		m:      m,
		log:    l,
		tracer: tracer,
	}

	r.Mount("/", Handler(ret))
	swg, err := GetSwagger()
	if err != nil {
		//log.Fatal("swagger fail")
		ret.log.WithField("stacktrace", string(debug.Stack())).Panicf("swagger fail")
	}

	r.Get("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)
		_ = enc.Encode(swg)
	})

	//Added Prometheus
	r.Handle("/metrics", promhttp.Handler())

	ret.Mux = r

	return ret
}

type UrlData apichi.ApiUrlData

func (UrlData) Bind(r *http.Request) error {
	return nil
}

func (UrlData) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// (GET /getData/{adminURL})
func (rt *OpenApiChi) AdminRedirect(w http.ResponseWriter, r *http.Request, adminURL string) {
	span, ctx := opentracing.StartSpanFromContextWithTracer(r.Context(), rt.tracer,
		"AdminRedirect")
	defer span.Finish()

	span.LogFields(
		tracelog.String("adminURL", adminURL),
	)

	rt.log.WithField("method", r.Method).Info("AdminRedirect called")

	urldata := UrlData{
		AdminURL: adminURL,
	}

	nud, ipdata, err := rt.hs.GetDataHandle(ctx, apichi.ApiUrlData(urldata))
	if err != nil {
		rt.log.WithField("method", r.Method).Error(err)
		err = render.Render(w, r, apichi.ErrRender(err))
		if err != nil {
			rt.log.WithField("method", r.Method).Error(err)
		}
	}

	DataURLVars := PageVars{
		Data:     nud.Data,
		ShortURL: nud.ShortURL,
		IPData:   ipdata,
	}

	t, err := template.ParseFS(tpls, "pages/getData.html")
	if err != nil {
		rt.log.WithField("method", r.Method).Error("template parsing error: ", err)
	}
	err = t.Execute(w, DataURLVars)
	if err != nil {
		rt.log.WithField("method", r.Method).Error("template executing error: ", err)
	}

}

// (POST /shortenURL)
func (rt *OpenApiChi) GenShortURL(w http.ResponseWriter, r *http.Request) {
	span, ctx := opentracing.StartSpanFromContextWithTracer(r.Context(), rt.tracer,
		"AdminRedirect")
	defer span.Finish()

	rt.log.WithField("method", r.Method).Info("GenShortUrl called")

	err := r.ParseForm()
	if err != nil {
		rt.log.WithField("method", r.Method).Error("error parsing form:", err)
	}

	fullurl := r.Form.Get("fullurl")
	if fullurl == "" {
		rt.log.WithField("method", r.Method).Error("search query not found:", err)
	}

	urldata := UrlData{
		FullURL: fullurl,
	}

	nud, err := rt.hs.GenShortUrlHandle(ctx, apichi.ApiUrlData(urldata))
	if err != nil {
		rt.log.WithField("method", r.Method).Error(err)
		err = render.Render(w, r, apichi.ErrRender(err))
		if err != nil {
			rt.log.WithField("method", r.Method).Error(err)
		}
	}

	shortenURLVars := PageVars{
		ShortURL: nud.ShortURL,
		AdminURL: nud.AdminURL,
		FullURL:  nud.FullURL,
	}

	defer func() {
		rt.m.fullUrlLengthHistogram.Set(float64(len(shortenURLVars.FullURL)))
	}()

	t, err := template.ParseFS(tpls, "pages/shortenURL.html")
	if err != nil {
		rt.log.WithField("method", r.Method).Error("template parsing error: ", err)
	}
	err = t.Execute(w, shortenURLVars)
	if err != nil {
		rt.log.WithField("method", r.Method).Error("template executing error: ", err)
	}

}

// (GET /su/{shortURL})
func (rt *OpenApiChi) Redirect(w http.ResponseWriter, r *http.Request, shortURL string) {
	span, ctx := opentracing.StartSpanFromContextWithTracer(r.Context(), rt.tracer,
		"AdminRedirect")
	defer span.Finish()

	span.LogFields(
		tracelog.String("shortURL", shortURL),
	)

	rt.log.WithField("method", r.Method).Info("Redirect called")

	if shortURL == "" {
		err := render.Render(w, r, apichi.ErrInvalidRequest(http.ErrNotSupported))
		rt.log.WithField("method", r.Method).Error(err)
		if err != nil {
			rt.log.WithField("method", r.Method).Error(err)
		}
		return
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		rt.log.WithField("method", r.Method).Error(err)
	}

	nud, err := rt.hs.RedirectionHandle(ctx, shortURL, ip)
	if err != nil {
		rt.log.WithField("method", r.Method).Error(err)
		err = render.Render(w, r, apichi.ErrRender(err))
		if err != nil {
			rt.log.WithField("method", r.Method).Error(err)
		}
		return
	}

	http.Redirect(w, r, nud.FullURL, http.StatusSeeOther)

}

// (GET /home)
func (rt *OpenApiChi) GetUserFullURL(w http.ResponseWriter, r *http.Request) {
	span, _ := opentracing.StartSpanFromContextWithTracer(r.Context(), rt.tracer,
		"AdminRedirect")
	defer span.Finish()

	rt.log.WithField("method", r.Method).Info("Homepage opened")

	t, err := template.ParseFS(tpls, "pages/homepage.html")
	if err != nil {
		rt.log.WithField("method", r.Method).Error("template parsing error: ", err)
	}

	err = t.Execute(w, nil)
	if err != nil {
		rt.log.WithField("method", r.Method).Error("template execute error: ", err)
	}
}
