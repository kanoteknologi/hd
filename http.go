package hd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"

	"git.kanosolution.net/kano/kaos"
	"git.kanosolution.net/kano/kaos/deployer"
	"github.com/sebarcode/codekit"
)

const DeployerName string = "kaos-http-deployer"

type httpDeployer struct {
	deployer.BaseDeployer
	mx *http.ServeMux

	isWrapError bool
	wrapErrFn   func(http.ResponseWriter, int, string)
}

func init() {
	deployer.RegisterDeployer(DeployerName, func() (deployer.Deployer, error) {
		return new(httpDeployer), nil
	})
}

// NewHttpDeployer initiate deployer to kaos
func NewHttpDeployer(fn func(http.ResponseWriter, int, string)) deployer.Deployer {
	dep := new(httpDeployer)
	if fn != nil {
		dep.SetWrapErrorFunction(fn)
	}
	return dep.SetThis(dep)
}

func (h *httpDeployer) PreDeploy(obj interface{}) error {
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

func (h *httpDeployer) Name() string {
	return DeployerName
}

func (h *httpDeployer) SetWrapErrorFunction(fn func(w http.ResponseWriter, status int, errTxt string)) {
	h.wrapErrFn = fn
	h.isWrapError = true
}

func (h *httpDeployer) DeployRoute(svc *kaos.Service, sr *kaos.ServiceRoute, obj interface{}) error {
	var ok bool
	h.mx, ok = obj.(*http.ServeMux)
	if !ok {
		return fmt.Errorf("second parameter should be a mux")
	}

	// path := svc.BasePoint() + sr.Path
	httpFn := func(w http.ResponseWriter, r *http.Request) {
		//ins := make([]reflect.Value, 2)
		//outs := make([]reflect.Value, 2)

		// create request
		ctx := kaos.NewContext(svc, sr)
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
					runErrTxt = fmt.Sprintf("error when running requested operation, please contact system admin and give this number [%d]", randNo)
					ctx.Log().Error(fmt.Sprintf("[%d] %v trace: %s", randNo, r, string(debug.Stack())))
				}
			}()
			bs, err = ioutil.ReadAll(r.Body)
			defer r.Body.Close()
			if tmp, err = h.This().Byter().Decode(bs, tmp, nil); err != nil {
				runErrTxt = "unable to get payload: " + err.Error()
				return
			}
		}()

		if runErrTxt != "" {
			statusCode := ctx.Data().Get("http_status_code", http.StatusInternalServerError).(int)
			if h.isWrapError {
				h.wrapErrFn(w, statusCode, runErrTxt)
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
					runErrTxt = fmt.Sprintf("error when running requested operation, please contact system admin and give this number [%d]", randNo)
					ctx.Log().Error(fmt.Sprintf("[%d] %v trace: %s", randNo, r, string(debug.Stack())))
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
			if h.isWrapError {
				h.wrapErrFn(w, statusCode, runErrTxt)
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
		bs, err = h.This().Byter().Encode(res)
		if err != nil {
			statusCode := ctx.Data().Get("http_status_code", http.StatusInternalServerError).(int)
			errTxt := "unable to encode output: " + err.Error()
			if h.isWrapError {
				h.wrapErrFn(w, statusCode, errTxt)
				return
			}
			w.WriteHeader(statusCode)
			w.Write([]byte(errTxt))
			return
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

	sr.Path = strings.ReplaceAll(sr.Path, "\\", "/")
	svc.Log().Infof("registering to mux: %s", sr.Path)
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
	r := ctx.Data().Get("http-request", nil)
	w := ctx.Data().Get("http-writer", nil)

	if _, ok := r.(*http.Request); ok {
		if _, ok = w.(http.ResponseWriter); ok {
			return ok
		}
	}
	return false
}
