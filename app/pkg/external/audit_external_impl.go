package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Praisindo/pkg-library/app/pkg/exception"
	"github.com/Praisindo/pkg-library/app/pkg/httputils"
	"github.com/sirupsen/logrus"
)

type AuditSvcExternalImpl struct {
	AuditSvcURL    string
	AuditSvcAPIKey string
}

func NewAuditSvcExternalImpl(
	config *AuditSvcExternalImpl,
) AuditSvcExternal {
	return &AuditSvcExternalImpl{
		AuditSvcURL:    config.AuditSvcURL,
		AuditSvcAPIKey: config.AuditSvcAPIKey,
	}
}

func (e *AuditSvcExternalImpl) CreateAudit(ctx context.Context, model *RequestAuditCreate) *exception.Exception {
	payload, err := json.Marshal(model)
	if err != nil {
		logrus.Error("error when marshal payload", "error", err)
		return exception.Internal("error marshalling old value", err)
	}

	headers := map[string]any{
		"Authorization": "Bearer " + e.AuditSvcAPIKey,
	}
	res, err := httputils.DoHttpRequest(ctx, "POST", e.AuditSvcURL, headers, payload)
	if err != nil {
		logrus.Error("error when sending request", "error", err)
		return exception.Internal("error ", err)
	}

	if res.StatusCode != http.StatusOK {
		logrus.Error(fmt.Sprintf("failed to send audit request of %s", model.RequestUUID))
		return exception.Internal(fmt.Sprintf("failed to send audit request of %s", model.RequestUUID), err)
	}

	logrus.Info("success send audit request", model.RequestUUID)
	return nil
}
