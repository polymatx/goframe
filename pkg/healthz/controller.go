package healthz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/polymatx/goframe/pkg/framework"
	"github.com/polymatx/goframe/pkg/framework/controller"
	"github.com/polymatx/goframe/pkg/framework/router"
	"github.com/polymatx/goframe/pkg/xlog"
	"github.com/sirupsen/logrus"
)

type route struct {
}

func (r route) check(w http.ResponseWriter, rq *http.Request) {
	lock.RLock()
	defer lock.RUnlock()

	var (
		errs []error
	)

	for i := range all {
		if err := all[i].Health(rq.Context()); err != nil {
			logrus.Error(err)
			errs = append(errs, err)
		}
	}

	w.Header().Set("time", time.Now().String())
	if len(errs) > 0 {
		xlog.GetWithError(context.Background(), errors.New("health failed")).Error(func() string {
			res := ""
			for i := range errs {
				res += " " + errs[i].Error()
			}
			return res
		}())
		w.WriteHeader(http.StatusInternalServerError)
		for i := range errs {
			fmt.Fprint(w, errs[i].Error())
		}
		return
	}
	_ = framework.JSON(w, http.StatusOK, struct {
		Time string `json:"time"`
	}{
		Time: time.Now().String(),
	})
}

func (r route) Routes(m *mux.Router) {
	m.Handle("/healthz", controller.Mix(r.check)).Methods("GET", "HEAD")
}

func RegisterRoute() {
	router.Register(&route{})
}
