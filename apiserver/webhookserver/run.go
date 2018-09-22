package webhookserver

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	dgsv1alpha1 "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/apis/azuregaming/v1alpha1"
	dgsscheme "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/client/clientset/versioned/scheme"
	shared "github.com/dgkanatsios/azuregameserversscalingkubernetes/pkg/shared"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	defaulter = runtime.ObjectDefaulter(runtimeScheme)

	podLabels = map[string]string{
		shared.LabelIsDedicatedGameServer: "true",
	}
)

var server WebhookServer

type WebhookServer struct {
	http.Server
}

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
	// defaulting with webhooks:
	// https://github.com/kubernetes/kubernetes/issues/57982
	_ = v1.AddToScheme(runtimeScheme)
	dgsscheme.AddToScheme(dgsscheme.Scheme)
}

// the Path of the JSON patch is a JSON pointer value
// so we need to escape any "/"s in the key we add to the annotation
// https://tools.ietf.org/html/rfc6901
func escapeJSONPointer(s string) string {
	esc := strings.Replace(s, "~", "~0", -1)
	esc = strings.Replace(esc, "/", "~1", -1)
	return esc
}

// main mutation process
func (whsvr *WebhookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request

	isDGSCol := false
	isDGS := false
	hasExistingAffinity := false

	var dgsCol dgsv1alpha1.DedicatedGameServerCollection
	err := json.Unmarshal(req.Object.Raw, &dgsCol)
	if err == nil {
		isDGSCol = true
		hasExistingAffinity = dgsCol.Spec.Template.Affinity != nil
	}

	var dgs dgsv1alpha1.DedicatedGameServer
	err = json.Unmarshal(req.Object.Raw, &dgs)
	if err == nil {
		isDGS = true
		hasExistingAffinity = dgsCol.Spec.Template.Affinity != nil
	}

	if !isDGSCol && !isDGS {
		log.Errorf("Could not unmarshal raw object to either DGSCol or DGS: %v", err)
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	log.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v k8sOperation=%v UserInfo=%v",
		req.Kind, req.Namespace, req.Name, dgsCol.Name, req.UID, req.Operation, req.UserInfo)

	var patch []patchOperation
	patch = append(patch, addAffinity(hasExistingAffinity))

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		log.Errorf("Error during marshaling: %v", err.Error())
		return &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	}

	jsonPatchType := v1beta1.PatchTypeJSONPatch

	log.Infof("AdmissionResponse: patch=%v", string(patchBytes))
	return &v1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &jsonPatchType,
	}
}

// Serve method for webhook server
func (whsvr *WebhookServer) serve(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		log.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Errorf("Content-Type=%s, expect application/json", contentType)
		http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
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
		admissionResponse = whsvr.mutate(&ar)
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
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}
	log.Infof("Ready to write reponse ...")
	if _, err := w.Write(resp); err != nil {
		log.Errorf("Can't write response: %v", err)
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}

func Run(certFile, keyFile string, port int) *WebhookServer {
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		log.Errorf("Failed to load key pair: %v", err)
	}

	whsvr := &WebhookServer{
		http.Server{
			Addr:      fmt.Sprintf(":%v", port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", whsvr.serve)
	whsvr.Handler = mux

	log.Printf("WebHook Server waiting for requests at port %d", port)

	// start webhook server in new rountine
	go func() {
		if err := whsvr.ListenAndServeTLS(certFile, keyFile); err != nil {
			log.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	return whsvr
}

func addAffinity(affinityExists bool) patchOperation {
	affinity := corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				corev1.WeightedPodAffinityTerm{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: podLabels,
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}

	operation := "add"
	if affinityExists {
		operation = "replace"
	}

	patchAffinity := patchOperation{
		Op:    operation,
		Path:  "/spec/template/affinity",
		Value: affinity,
	}

	return patchAffinity
}
