package main

import (
	"encoding/json"
	"fmt"
	"github.com/navikt/mutatingflow/pkg/commons"
	"github.com/navikt/mutatingflow/pkg/mutates/notebooks"
	"github.com/navikt/mutatingflow/pkg/mutates/pipelines"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"net/http"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

type WebhookServer struct {
	server     *http.Server
	parameters commons.Parameters
}

func (server *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	request := ar.Request
	switch request.Kind.Kind {
	case "Notebook":
		return notebooks.MutateNotebook(request, server.parameters)
	case "Pod":
		return pipelines.MutatePod(request)
	}

	return &v1beta1.AdmissionResponse{
		Allowed: false,
		Result: &metav1.Status{
			Message: fmt.Sprintf("unknown resource: '%s'", ar.Kind),
		},
	}
}

func (server *WebhookServer) serve(responseWriter http.ResponseWriter, request *http.Request) {
	var body []byte
	if request.Body != nil {
		if data, err := ioutil.ReadAll(request.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		log.Error("empty body")
		http.Error(responseWriter, "empty body", http.StatusBadRequest)
		return
	}

	contentType := request.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Errorf("Content-Type=%s, expected application/json", contentType)
		http.Error(responseWriter, "invalid Content-Type, expected `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		log.Errorf("Can't decode body: %v", err)
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		admissionResponse = server.mutate(&ar)
	}

	admissionReview := v1beta1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
		}
	}

	resp, err := json.Marshal(admissionReview)
	if err != nil {
		log.Errorf("Can't encode response: %v", err)
		http.Error(responseWriter, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}

	if _, err := responseWriter.Write(resp); err != nil {
		log.Errorf("Can't write response: %v", err)
		http.Error(responseWriter, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
