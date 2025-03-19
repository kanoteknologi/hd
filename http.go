package hd

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"

	"git.kanosolution.net/kano/kaos"
	"git.kanosolution.net/kano/kaos/deployer"
	"github.com/sebarcode/codekit"
)

const DeployerName string = "kaos-http-deployer"

type HttpDeployer struct {
	deployer.BaseDeployer
	mx *http.ServeMux

	isWrapError bool
	wrapErrFn   func(*kaos.Context, string)
}

func init() {
	deployer.RegisterDeployer(DeployerName, func(obj interface{}) (deployer.Deployer, error) {
		return new(HttpDeployer), nil
	})
}

// NewHttpDeployer initiate deployer to kaos
func NewHttpDeployer(fn func(*kaos.Context, string)) *HttpDeployer {
	dep := new(HttpDeployer)
	if fn != nil {
		dep.SetWrapErrorFunction(fn)
	}
	dep.SetThis(dep)
	return dep
}

func (h *HttpDeployer) PreDeploy(obj interface{}) error {
	var ok bool
	h.mx, ok = obj.(*http.ServeMux)
	if !ok {
		return fmt.Errorf("second parameter should be a mux")
	}
	h.mx.Handle("/beat", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	return nil
}

func (h *HttpDeployer) Name() string {
	return DeployerName
}

func (h *HttpDeployer) SetWrapErrorFunction(fn func(ctx *kaos.Context, errTxt string)) {
	h.wrapErrFn = fn
	h.isWrapError = true
}

func (h *HttpDeployer) Fn(svc *kaos.Service, sr *kaos.ServiceRoute) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		//ins := make([]reflect.Value, 2)
		//outs := make([]reflect.Value, 2)

		// create request
		ctx := kaos.NewContextFromService(svc, sr)
		ctx.Data().Set("path", sr.Path)
		ctx.Data().Set("http_request", r)
		ctx.Data().Set("http_writer", w)

		// get request type
		var fn1Type reflect.Type
		if sr.RequestType == nil {
			fn1Type = sr.Fn.Type().In(1)
		} else {
			fn1Type = sr.RequestType
		}
		p1IsPtr := fn1Type.String()[0] == '*'

		// create new request buffer with same type of fn1type,
		// it need to be pointerized first
		var tmp interface{}
		//var tmpType reflect.Type
		if p1IsPtr {
			tmpv := reflect.New(fn1Type.Elem())
			tmp = tmpv.Interface()
			//tmpType = tmpv.Type()
		} else {
			tmpv := reflect.New(fn1Type)
			tmp = tmpv.Elem().Interface()
			//tmpType = tmpv.Elem().Type()
		}
		//fmt.Printf("\nBefore tmp: %T %s\n", tmp, codekit.JsonString(tmp))

		// get request
		var err error
		var bs []byte
		var runErrTxt string
		func() {
			defer func() {
				if r := recover(); r != nil {
					randNo := codekit.RandInt(999999)
					runErrTxt = fmt.Sprintf("error when running requested operation %s, please contact system admin and give this number [%d]", sr.Path, randNo)
					ctx.Log().Error(codekit.JsonString(
						codekit.M{}.
							Set("path", sr.Path).
							Set("status", http.StatusInternalServerError).
							Set("error", runErrTxt).
							Set("trace", string(debug.Stack()))))
				}
			}()
			bs, err = io.ReadAll(r.Body)
			defer r.Body.Close()
			if tmp, err = h.This().Byter().Decode(bs, tmp, nil); err != nil {
				runErrTxt = "unable to get payload: " + err.Error()
				return
			}
		}()

		if runErrTxt != "" {
			statusCode := ctx.Data().Get("http_status_code", http.StatusInternalServerError).(int)
			ctx.Log().Error(codekit.JsonString(
				codekit.M{}.
					Set("path", sr.Path).
					Set("status", statusCode).
					Set("msg", runErrTxt)))
			if h.isWrapError {
				h.wrapErrFn(ctx, runErrTxt)
				return
			}
			w.WriteHeader(statusCode)
			w.Write([]byte(runErrTxt))
			return
		}
		//fmt.Printf("\nAfter tmp: %T %s\n", tmp, codekit.JsonString(tmp))

		// run the function
		var res interface{}
		runErrTxt = ""
		func() {
			defer func() {
				if r := recover(); r != nil {
					randNo := codekit.RandInt(999999)
					runErrTxt = fmt.Sprintf("error when running requested operation %s, please contact system admin and give this number [%d]", sr.Path, randNo)
					ctx.Log().Error(codekit.JsonString(
						codekit.M{}.
							Set("path", sr.Path).
							Set("status", http.StatusInternalServerError).
							Set("error", runErrTxt).
							Set("trace", string(debug.Stack()))))
					ctx.Data().Get("http_status_code", http.StatusInternalServerError)
				}
			}()

			res, err = sr.Run(ctx, svc, tmp)
			if err != nil {
				runErrTxt = err.Error()
				return
			}
		}()

		if runErrTxt != "" {
			statusCode := ctx.Data().Get("http_status_code", http.StatusBadRequest).(int)
			ctx.Log().Error(codekit.JsonString(
				codekit.M{}.
					Set("path", sr.Path).
					Set("status", statusCode).
					Set("msg", runErrTxt)))
			if h.isWrapError {
				h.wrapErrFn(ctx, runErrTxt)
				return
			}
			w.WriteHeader(statusCode)
			w.Write([]byte(runErrTxt))
			return
		}

		if ctx.Data().Get("kaos_command_1", "").(string) == "stop" {
			return
		}

		// encode output
		//svc.Log().Infof("data: %v err: %v\n", res, err)
		noEncode := ctx.Data().Get("no_encode", "").(string) == "1"
		if !noEncode {
			bs, err = h.This().Byter().Encode(res)
			if err != nil {
				statusCode := ctx.Data().Get("http_status_code", http.StatusInternalServerError).(int)
				ctx.Log().Error(codekit.JsonString(
					codekit.M{}.
						Set("path", sr.Path).
						Set("status", statusCode).
						Set("msg", runErrTxt)))
				errTxt := "unable to encode output: " + err.Error()
				if h.isWrapError {
					h.wrapErrFn(ctx, errTxt)
					return
				}
				w.WriteHeader(statusCode)
				w.Write([]byte(errTxt))
				return
			}
		} else {
			bs = res.([]byte)
		}

		//-- status code
		statusCode := ctx.Data().Get("http_status_code", http.StatusOK).(int)
		w.WriteHeader(statusCode)

		//-- content type
		if contentType := ctx.Data().Get("http_content_type", "").(string); contentType != "" {
			w.Header().Add("Content-Type", contentType)
		}

		//-- headers
		headers := ctx.Data().Get("http_headers", map[string]string{}).(map[string]string)
		for k, h := range headers {
			w.Header().Add(k, h)
		}

		w.Write(bs)
	}
}

func (h *HttpDeployer) DeployRoute(svc *kaos.Service, sr *kaos.ServiceRoute, obj interface{}) error {
	var ok bool
	h.mx, ok = obj.(*http.ServeMux)
	if !ok {
		return fmt.Errorf("second parameter should be a mux")
	}

	httpFn := h.Fn(svc, sr)
	sr.Path = strings.ReplaceAll(sr.Path, "\\", "/")
	svc.Log().Infof("registering to mux-rest: %s", sr.Path)
	h.mx.Handle(sr.Path, http.HandlerFunc(httpFn))
	return nil
}

func SetStatusCode(ctx *kaos.Context, statusCode int) {
	ctx.Data().Set("http_status_code", statusCode)
}

func SetHeaders(ctx *kaos.Context, headers map[string]string) {
	ctx.Data().Set("http_headers", headers)
}

func SetContentType(ctx *kaos.Context, contentType string) {
	ctx.Data().Set("http_content_type", contentType)
}

func IsHttpHandler(ctx *kaos.Context) bool {
	r := ctx.Data().Get("http_request", nil)
	w := ctx.Data().Get("http_writer", nil)

	if _, ok := r.(*http.Request); ok {
		if _, ok = w.(http.ResponseWriter); ok {
			return ok
		}
	}
	return false
}

func (h *HttpDeployer) Activate(obj interface{}) error {
	mux := obj.(*http.ServeMux)
	if mux == nil {
		return fmt.Errorf("mux is nil")
	}
	host, ok := h.Get("host").(string)
	if !ok || host == "" {
		return fmt.Errorf("config \"Host\" is not set")
	}
	go http.ListenAndServe(host, mux)
	return nil
}
