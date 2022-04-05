package openapichi

import (
	"io"

	opentracing "github.com/opentracing/opentracing-go"

	log "github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

type JaegerUtils struct {
	reporter jaeger.Reporter
	closer   io.Closer
}

type LogrusAdapter struct {
	logger *log.Logger
}

func (l LogrusAdapter) Error(msg string) {
	l.logger.Errorf(msg)
}

func (l LogrusAdapter) Infof(msg string, args ...interface{}) {
	l.logger.Infof(msg, args...)
}

func (ju *JaegerUtils) InitJaeger(url string, service string, logger *log.Logger) (opentracing.Tracer, io.Closer, error) {
	transport, err := jaeger.NewUDPTransport(url, 0)
	if err != nil {
		log.Errorln(err.Error())
	}

	logAdapt := LogrusAdapter{logger: logger}

	reporter := jaeger.NewCompositeReporter(
		jaeger.NewLoggingReporter(logAdapt),
		jaeger.NewRemoteReporter(transport,
			jaeger.ReporterOptions.Logger(logAdapt),
		),
	)
	//defer reporter.Close()

	cfg := &config.Configuration{
		ServiceName: service,
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}

	tracer, closer, err := cfg.NewTracer(config.Reporter(reporter))
	if err != nil {
		return nil, nil, err
	}
	//defer closer.Close()

	ju.reporter = reporter
	ju.closer = closer

	return tracer, closer, nil
}

func (ju *JaegerUtils) Close() {
	ju.reporter.Close()
	ju.closer.Close()
}
