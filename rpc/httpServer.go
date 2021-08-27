package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/greboid/irc-bot/v5/irc"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type httpServer struct {
	WebPort int
	plugins []Plugin
	pathMap map[string]*descriptor
	logger  irc.Logger
}

func (h *httpServer) mustEmbedUnimplementedHTTPPluginServer() {
}

type descriptor struct {
	prefix  string
	stream  *HTTPPlugin_GetRequestServer
	receive chan *HttpResponse
}

func NewHttpServer(port int, plugin []Plugin, logger irc.Logger) *httpServer {
	return &httpServer{
		WebPort: port,
		plugins: plugin,
		pathMap: make(map[string]*descriptor),
		logger:  logger,
	}

}

func (h *httpServer) Start() {
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/", h.handleRequest)
		server := http.Server{
			Addr:    fmt.Sprintf(":%d", h.WebPort),
			Handler: mux,
		}
		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				h.logger.Errorf("Error starting HTTP: %s", err.Error())
			}
		}()
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, os.Kill)
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			h.logger.Errorf("Unable to shutdown: %s", err.Error())
		}
	}()
}

func (h *httpServer) authPlugin(ctx context.Context) (context.Context, error) {
	token, err := grpcauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %s", err.Error())
	}
	if !h.checkPlugin(token) {
		return nil, status.Errorf(codes.Unauthenticated, "access denied")
	}
	return ctx, nil
}

func (h *httpServer) checkPlugin(token string) bool {
	for _, plugin := range h.plugins {
		if plugin.Token == token {
			return true
		}
	}
	return false
}

func (h *httpServer) handleRequest(writer http.ResponseWriter, request *http.Request) {
	for k := range h.pathMap {
		if strings.HasPrefix(request.URL.Path, fmt.Sprintf("/%s", h.pathMap[k].prefix)) {
			stream := *h.pathMap[k].stream
			if stream != nil {
				rpcHttpc, err := ConvertHTTPToRPC(request)
				if err != nil {
					h.logger.Errorf("Unable to read input")
					writer.WriteHeader(http.StatusInternalServerError)
					_, _ = writer.Write([]byte("Unable to read input"))
					return
				}
				err = stream.Send(rpcHttpc)
				if err != nil {
					h.logger.Errorf("Unable to send to plugin")
					writer.WriteHeader(http.StatusInternalServerError)
					_, _ = writer.Write([]byte("Unable to send to handler"))
					return
				}
				select {
				case response := <-h.pathMap[k].receive:
					for index := range response.Header {
						writer.Header().Add(response.Header[index].Key, response.Header[index].Value)
					}
					writer.WriteHeader(int(response.Status))
					_, _ = writer.Write(response.Body)
					return
				case <-time.After(5 * time.Second):
					h.logger.Errorf("Timeout waiting for plugin: %s", request.URL.Path)
					writer.WriteHeader(http.StatusGatewayTimeout)
					_, _ = writer.Write([]byte("Timeout waiting for handler"))
					return
				}
			}
		}
	}
	writer.WriteHeader(http.StatusNotFound)
	_, _ = writer.Write([]byte("Handler not found"))
}

func (h *httpServer) GetRequest(stream HTTPPlugin_GetRequestServer) error {
	path := metautils.ExtractIncoming(stream.Context()).Get("path")
	if _, ok := h.pathMap[path]; ok {
		return errors.New("prefix already registered")
	}
	h.logger.Debugf("Plugin listening for /%s/*", path)
	h.pathMap[path] = &descriptor{
		prefix:  path,
		receive: make(chan *HttpResponse, 1),
		stream:  &stream,
	}
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			h.logger.Debugf("Plugin stopped listening for /%s/*", path)
			delete(h.pathMap, path)
			return nil
		}
		if err != nil {
			h.logger.Debugf("Plugin stopped listening for /%s/*", path)
			delete(h.pathMap, path)
			return err
		}
		h.pathMap[path].receive <- in
	}
}

func ConvertHTTPToRPC(r *http.Request) (*HttpRequest, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	return &HttpRequest{
		Header:               ConvertToRPCHeaders(r.Header),
		Body:                 body,
		Path:                 r.URL.Path,
		Method:               r.Method,
	}, nil
}

func ConvertRPCToHTTP(r *HttpRequest) *http.Request {
	return &http.Request{
		Method:           r.Method,
		Header:           ConvertFromRPCHeaders(r.Header),
		Body:             io.NopCloser(bytes.NewReader(r.Body)),
	}
}

func ConvertFromRPCHeaders(headers []*HttpHeader) http.Header {
	httpHeaders := http.Header{}
	for index := range headers {
		httpHeaders.Add(headers[index].Key, headers[index].Value)
	}
	return httpHeaders
}

func ConvertToRPCHeaders(headers http.Header) []*HttpHeader {
	rpcHeaders := make([]*HttpHeader, 0)
	for key, value := range headers {
		for index := range value {
			rpcHeaders = append(rpcHeaders, &HttpHeader{
				Key:   key,
				Value: value[index],
			})
		}

	}
	return rpcHeaders
}
