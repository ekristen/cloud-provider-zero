package webhook

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/ekristen/cloud-provider-zero/pkg/webhook/resources/node"
	"github.com/gorilla/mux"
	"github.com/rancher/dynamiclistener"
	dlserver "github.com/rancher/dynamiclistener/server"
	"github.com/rancher/wrangler/pkg/apply"
	core "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/webhook"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const (
	serviceName      = "cpz-webhook"
	certName         = "cpz-webhook-tls"
	caName           = "cpz-webhook-ca"
	webhookHTTPPort  = 0 // value of 0 indicates we do not want to use http.
	webhookHTTPSPort = 9443
)

var (
	// These strings have to remain as vars since we need the address below.
	validationPath              = "/v1/webhook/validation"
	mutationPath                = "/v1/webhook/mutation"
	clientPort                  = int32(9443)
	clusterScope                = admissionv1.ClusterScope
	namespaceScope              = admissionv1.NamespacedScope
	failPolicyFail              = admissionv1.Fail
	failPolicyIgnore            = admissionv1.Ignore
	sideEffectClassNone         = admissionv1.SideEffectClassNone
	sideEffectClassNoneOnDryRun = admissionv1.SideEffectClassNoneOnDryRun
)

var tlsOpt = func(config *tls.Config) *tls.Config {
	config.MinVersion = tls.VersionTLS12
	config.CurvePreferences = []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256}
	config.CipherSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	}

	return config
}

type Options struct {
	DevelopmentBaseURL string
	Namespace          string
}

func NewOptions() *Options {
	return &Options{
		Namespace: "cloud-provider-zero",
	}
}

func Start(ctx context.Context, apply apply.Apply, secrets core.SecretController, options *Options) error {
	apply = apply.WithDynamicLookup()

	secrets.OnChange(ctx, "secrets", func(key string, secret *corev1.Secret) (*corev1.Secret, error) {
		if secret == nil || secret.Name != caName || secret.Namespace != options.Namespace || len(secret.Data[corev1.TLSCertKey]) == 0 {
			return nil, nil
		}

		mutationClientConfig := admissionv1.WebhookClientConfig{
			Service: &admissionv1.ServiceReference{
				Namespace: options.Namespace,
				Name:      serviceName,
				Path:      &mutationPath,
				Port:      &clientPort,
			},
			CABundle: secret.Data[corev1.TLSCertKey],
		}

		if options != nil && options.DevelopmentBaseURL != "" {
			logrus.Warn("development mode, setting URL")
			mutationUrl := fmt.Sprintf("%s/%s", strings.TrimRight(options.DevelopmentBaseURL, "/"), mutationPath)
			mutationClientConfig.Service = nil
			mutationClientConfig.URL = &mutationUrl
		}

		return secret, apply.WithOwner(secret).ApplyObjects(&admissionv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cloud-provider-zero",
			},
			Webhooks: []admissionv1.MutatingWebhook{
				{
					Name:                    "cpz.ekristen.dev",
					ClientConfig:            mutationClientConfig,
					Rules:                   rules,
					FailurePolicy:           &failPolicyIgnore,
					SideEffects:             &sideEffectClassNone,
					AdmissionReviewVersions: []string{"v1"},
				},
			},
		})
	})

	tlsName := fmt.Sprintf("%s.%s.svc", serviceName, options.Namespace)

	tlsConfig := tlsOpt(&tls.Config{
		ServerName: tlsName,
	})

	mutationRouter := webhook.NewRouter()
	mutationRouter.Kind("Node").Group(corev1.GroupName).Type(&corev1.Node{}).Handle(node.NewMutation())

	router := mux.NewRouter()
	router.Handle(mutationPath, mutationRouter)

	err := dlserver.ListenAndServe(ctx, webhookHTTPSPort, webhookHTTPPort, router, &dlserver.ListenOpts{
		CANamespace:   options.Namespace,
		CAName:        caName,
		CertNamespace: options.Namespace,
		CertName:      certName,
		Secrets:       secrets,
		TLSListenerConfig: dynamiclistener.Config{
			CN:           tlsName,
			Organization: []string{"ekristen.dev"},
			TLSConfig:    tlsConfig,
		},
	})
	if err != nil {
		logrus.WithError(err).Fatalf("listen: %s\n", err)
	}

	logrus.WithField("port", webhookHTTPSPort).Info("starting webhook api server")

	return nil
}
