package hd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"runtime/debug"
	"strings"
	"syscall"

	"git.kanosolution.net/kano/kaos"
	"git.kanosolution.net/kano/kaos/deployer"
	"github.com/eaciit/toolkit"
)

const DeployerName string = "kaos-http-deployer"

type httpDeployer struct {
	deployer.BaseDeployer
	mx *http.ServeMux
}

func init() {
	deployer.RegisterDeployer(DeployerName, func() (deployer.Deployer, error) {
		return new(httpDeployer), nil
	})
}

// NewHttpDeployer initiate deployer to kaos
func NewHttpDeployer() deployer.Deployer {
	dep := new(httpDeployer)
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
		ctx.Data().Set("http-request", r)
		ctx.Data().Set("http-writer", w)
		authTxt := r.Header.Get("Authorization")
		if strings.HasPrefix(authTxt, "Bearer ") {
			ctx.Data().Set("jwt-token", strings.Replace(authTxt, "Bearer ", "", 1))
		}

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
		//fmt.Printf("\nBefore tmp: %T %s\n", tmp, toolkit.JsonString(tmp))

		// get request
		var err error
		var bs []byte
		var runErrTxt string
		func() {
			defer func() {
				if r := recover(); r != nil {
					randNo := toolkit.RandInt(999999)
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
			statusCode := ctx.Data().Get("http_status_code", http.StatusBadRequest).(int)
			w.WriteHeader(statusCode)
			w.Write([]byte(runErrTxt))
			return
		}
		//fmt.Printf("\nAfter tmp: %T %s\n", tmp, toolkit.JsonString(tmp))

		// run the function
		var res interface{}
		runErrTxt = ""
		func() {
			defer func() {
				if r := recover(); r != nil {
					randNo := toolkit.RandInt(999999)
					runErrTxt = fmt.Sprintf("error when running requested operation, please contact system admin and give this number [%d]", randNo)
					ctx.Log().Error(fmt.Sprintf("[%d] %v trace: %s", randNo, r, string(debug.Stack())))
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
			w.WriteHeader(statusCode)
			w.Write([]byte(runErrTxt))
			return
		}

		if ctx.Data().Get("kaos-command-1", "").(string) == "stop" {
			return
		}

		// encode output
		//svc.Log().Infof("data: %v err: %v\n", res, err)
		bs, err = h.This().Byter().Encode(res)
		if err != nil {
			statusCode := ctx.Data().Get("http_status_code", http.StatusBadRequest).(int)
			w.WriteHeader(statusCode)
			w.Write([]byte("unable to encode output: " + err.Error()))
			return
		}

		statusCode := ctx.Data().Get("http_status_code", http.StatusOK).(int)
		w.WriteHeader(statusCode)
		w.Write(bs)
	}

	sr.Path = strings.ReplaceAll(sr.Path, "\\", "/")
	svc.Log().Infof("registering to mux: %s", sr.Path)
	h.mx.Handle(sr.Path, http.HandlerFunc(httpFn))
	return nil
}

func StartKaosWebServer(s *kaos.Service, serviceName, hostName string, mux *http.ServeMux) (chan os.Signal, error) {
	var e error

	csign := make(chan os.Signal)

	// deploy
	if mux == nil {
		mux = http.NewServeMux()
	}

	if e = NewHttpDeployer().Deploy(s, mux); e != nil {
		s.Log().Errorf("unable to deploy. %s", e.Error())
		return csign, e
	}

	go func() {
		s.Log().Infof("Running %v service on %s", serviceName, hostName)
		err := http.ListenAndServe(hostName, mux)
		if err != nil {
			s.Log().Infof("error starting web server %s. %s", hostName, err.Error())
			csign <- syscall.SIGINT
		}
	}()

	return csign, nil
}
